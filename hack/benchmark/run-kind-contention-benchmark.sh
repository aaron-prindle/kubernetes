#!/usr/bin/env bash
# run-kind-contention-benchmark.sh
# End-to-end contention benchmark using a live Kubernetes cluster.
# It measures mutex contention and CPU overhead during highly parallel API Server LIST requests.

set -euo pipefail

CLUSTER_NAME="contention-bench-cluster"
IMAGE_NAME="contention-bench-node:latest"
REPLICAS=${1:-1000}
CONCURRENCY=${2:-50}

# Create output directories
PROFILES_DIR="$(pwd)/hack/benchmark/profiles"
mkdir -p "$PROFILES_DIR"

BRANCH_NAME=$(git rev-parse --abbrev-ref HEAD)
COMMIT_HASH=$(git rev-parse --short HEAD)
MUTEX_PROFILE="$PROFILES_DIR/mutex-${BRANCH_NAME//\//-}-${COMMIT_HASH}.prof"
CPU_PROFILE="$PROFILES_DIR/cpu-${BRANCH_NAME//\//-}-${COMMIT_HASH}.prof"

echo "=========================================================="
echo " Starting End-to-End API Server Contention Benchmark"
echo " Branch: $BRANCH_NAME"
echo " Commit: $COMMIT_HASH"
echo " Target Load: $REPLICAS Pending Pods"
echo " Concurrency: $CONCURRENCY parallel LIST clients"
echo "=========================================================="

echo "=> 1. Building Kubernetes Node Image from current tree..."
kind build node-image --image "$IMAGE_NAME"

echo "=> 2. Creating Kind Cluster..."
kind delete cluster --name "$CLUSTER_NAME" 2>/dev/null || true
kind create cluster --name "$CLUSTER_NAME" --image "$IMAGE_NAME"

echo "=> 3. Setting up proxy to API Server..."
kubectl proxy --port=8001 &
PROXY_PID=$!
trap "kill $PROXY_PID 2>/dev/null || true; kind delete cluster --name $CLUSTER_NAME 2>/dev/null || true" EXIT
sleep 2

echo "=> 4. Deploying load generator (Deployment with $REPLICAS Pending Pods)..."
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: contention-load-gen
spec:
  replicas: $REPLICAS
  selector:
    matchLabels:
      app: contention-load-gen
  template:
    metadata:
      labels:
        app: contention-load-gen
    spec:
      nodeSelector:
        non-existent-node: "true"
      containers:
      - name: pause
        image: registry.k8s.io/pause:3.9
EOF

echo "=> 5. Waiting for ReplicaSet to create $REPLICAS pods..."
while true; do
  CREATED=$(kubectl get pods -l app=contention-load-gen --no-headers 2>/dev/null | wc -l || echo 0)
  if [ "$CREATED" -ge "$REPLICAS" ]; then
    break
  fi
  echo "   Created $CREATED / $REPLICAS pods..."
  sleep 5
done

echo "=> 6. All pods created. Waiting 5 seconds for stabilization..."
sleep 5

echo "=> 7. Initiating $CONCURRENCY parallel LIST requests and capturing profiles..."

# We will capture CPU and Mutex profiles for 10 seconds.
curl -s "http://localhost:8001/debug/pprof/profile?seconds=10" > "$CPU_PROFILE" &
curl -s "http://localhost:8001/debug/pprof/mutex?seconds=10" > "$MUTEX_PROFILE" &

# Run highly parallel LIST requests for 10 seconds
# Using timeout so curl processes don't block indefinitely
timeout 12s bash -c "
  for i in \$(seq 1 $CONCURRENCY); do
    while true; do
      curl -m 5 -s http://localhost:8001/api/v1/pods > /dev/null || true
    done &
  done
  wait
" || true

echo "   Profiling complete."

echo "=========================================================="
echo "=> Top CPU Hotspots (flat):"
go tool pprof -top "$CPU_PROFILE" | head -n 15 || echo "   (No profile data found)"
echo "=========================================================="
echo " Benchmark Complete. Profiles saved. Cluster will now be deleted."
