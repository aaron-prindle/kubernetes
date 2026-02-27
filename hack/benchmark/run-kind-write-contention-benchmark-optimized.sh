#!/usr/bin/env bash
# run-kind-write-contention-benchmark-optimized.sh
# End-to-end contention benchmark focusing on parallel WRITES of NOVEL strings to trigger locking.

set -euo pipefail

CLUSTER_NAME="contention-bench-cluster"
IMAGE_NAME="contention-bench-node:latest"
REPLICAS=${1:-5000}
CONCURRENCY=${2:-50}

# Create output directories
PROFILES_DIR="$(pwd)/hack/benchmark/profiles"
mkdir -p "$PROFILES_DIR"

BRANCH_NAME=$(git rev-parse --abbrev-ref HEAD)
COMMIT_HASH=$(git rev-parse --short HEAD)
MUTEX_PROFILE="$PROFILES_DIR/write-mutex-${BRANCH_NAME//\//-}-${COMMIT_HASH}-novel.prof"
CPU_PROFILE="$PROFILES_DIR/write-cpu-${BRANCH_NAME//\//-}-${COMMIT_HASH}-novel.prof"

echo "=========================================================="
echo " Starting End-to-End API Server WRITE Contention Benchmark (NOVEL STRINGS)"
echo " Branch: $BRANCH_NAME"
echo " Commit: $COMMIT_HASH"
echo " Concurrency: $CONCURRENCY parallel clients"
echo "=========================================================="

echo "=> 1. Building Kubernetes Node Image from current tree..."
kind build node-image --image "$IMAGE_NAME"

echo "=> 2. Creating Kind Cluster with tuned API Server config..."
kind delete cluster --name "$CLUSTER_NAME" 2>/dev/null || true
kind create cluster --name "$CLUSTER_NAME" --image "$IMAGE_NAME" --config "$(pwd)/hack/benchmark/kind.yaml"

echo "=> 3. Setting up proxy to API Server..."
kubectl proxy --port=8001 &
PROXY_PID=$!
trap "kill $PROXY_PID 2>/dev/null || true; kind delete cluster --name $CLUSTER_NAME 2>/dev/null || true" EXIT
sleep 2

echo "=> 4. Initiating parallel WRITE requests with NOVEL strings..."

# We will capture CPU and Mutex profiles for 30 seconds.
curl -s "http://localhost:8001/debug/pprof/profile?seconds=30" > "$CPU_PROFILE" &
curl -s "http://localhost:8001/debug/pprof/mutex?seconds=30" > "$MUTEX_PROFILE" &

# Run highly parallel WRITE requests for 30 seconds
# KEY CHANGE: We are injecting RANDOM strings into the labels to force unique.Make
# to actually take the lock and insert the new string into its internal map on EVERY request.
timeout 32s bash -c "
  for i in \$(seq 1 $CONCURRENCY); do
    while true; do
      RAND1=\$RANDOM
      RAND2=\$RANDOM
      cat <<JSON | curl -m 5 -s -X PATCH -H 'Content-Type: application/apply-patch+yaml' http://localhost:8001/api/v1/namespaces/default/configmaps/cm-\$i-\$RAND1?fieldManager=bench-\$i > /dev/null || true
apiVersion: v1
kind: ConfigMap
metadata:
  name: cm-\$i-\$RAND1
  namespace: default
  labels:
    random-label: "val-\$RAND2-\$RAND1"
data:
  key1: value1
JSON
    done &
  done
  wait
" || true

echo "   Profiling complete."

echo "=========================================================="
echo "=> Top Mutex Delays (flat):"
go tool pprof -top -contentions "$MUTEX_PROFILE" | head -n 15 || echo "   (No profile data found)"
echo "=========================================================="
echo " Benchmark Complete. Profiles saved. Cluster will now be deleted."