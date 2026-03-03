#!/usr/bin/env bash
# run-kind-contention-benchmark.sh
# End-to-end contention benchmark using a live Kubernetes cluster.
# It measures mutex contention and CPU overhead during highly parallel API Server LIST requests.

set -euo pipefail

CLUSTER_NAME="contention-bench-cluster"
IMAGE_NAME="contention-bench-node:latest"
REPLICAS=${1:-10000}
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

echo "=> 2. Creating Kind Cluster with tuned API Server config..."
kind delete cluster --name "$CLUSTER_NAME" 2>/dev/null || true
kind create cluster --name "$CLUSTER_NAME" --image "$IMAGE_NAME" --config "$(pwd)/hack/benchmark/kind.yaml"

echo "=> 3. Extracting connection details..."
API_SERVER=$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')
kubectl create clusterrolebinding default-admin --clusterrole=cluster-admin --serviceaccount=default:default > /dev/null 2>&1 || true
TOKEN=$(kubectl create token default --duration=1h)
trap "kind delete cluster --name $CLUSTER_NAME 2>/dev/null || true" EXIT
sleep 2

echo "=> 4. Installing Kwok Controller..."
kubectl apply -f https://github.com/kubernetes-sigs/kwok/releases/download/v0.6.0/kwok.yaml
kubectl apply -f https://github.com/kubernetes-sigs/kwok/releases/download/v0.6.0/stage-fast.yaml
echo "   Waiting for Kwok Controller to be ready..."
sleep 15
kubectl -n kube-system wait --for=condition=Ready pods -l app=kwok-controller --timeout=300s

echo "=> 5. Creating 100 Fake Nodes using addnodes.sh..."
"$(pwd)/hack/benchmark/addnodes.sh" 100

echo "=> 6. Deploying load generator (StatefulSet with $REPLICAS Pods)..."
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: contention-load-gen
spec:
  podManagementPolicy: "Parallel"
  replicas: $REPLICAS
  selector:
    matchLabels:
      app: contention-load-gen
  template:
    metadata:
      labels:
        app: contention-load-gen
    spec:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: type
                operator: In
                values:
                - kwok
      tolerations:
      - key: "kwok.x-k8s.io/node"
        operator: "Exists"
        effect: "NoSchedule"
      containers:
      - name: pause
        image: registry.k8s.io/pause:3.9
EOF

echo "=> 7. Waiting for StatefulSet to create $REPLICAS pods..."
while true; do
  CREATED=$(kubectl get pods -l app=contention-load-gen --no-headers 2>/dev/null | wc -l || echo 0)
  RUNNING=$(kubectl get pods -l app=contention-load-gen --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l || echo 0)
  if [ "$RUNNING" -ge "$REPLICAS" ]; then
    break
  fi
  echo "   Created $CREATED / $REPLICAS pods ($RUNNING Running)..."
  sleep 5
done

echo "=> 8. All pods Running. Waiting 10 seconds for stabilization..."
sleep 10

echo "=> 9. Initiating $CONCURRENCY parallel LIST requests and capturing profiles..."

# We will capture CPU and Mutex profiles for 30 seconds.
curl -k -s -H "Authorization: Bearer $TOKEN" "$API_SERVER/debug/pprof/profile?seconds=30" > "$CPU_PROFILE" &
curl -k -s -H "Authorization: Bearer $TOKEN" "$API_SERVER/debug/pprof/mutex?seconds=30" > "$MUTEX_PROFILE" &

# Run highly parallel LIST requests for 30 seconds
# Using timeout so curl processes don't block indefinitely
timeout 32s bash -c "
  for i in \$(seq 1 $CONCURRENCY); do
    while true; do
      curl -k -H \"Authorization: Bearer $TOKEN\" -m 30 -s \"$API_SERVER/api/v1/pods\" > /dev/null || true
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
