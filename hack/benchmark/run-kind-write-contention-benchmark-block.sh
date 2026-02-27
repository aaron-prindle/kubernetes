#!/usr/bin/env bash
# run-kind-write-contention-benchmark-block.sh
# End-to-end contention benchmark capturing the BLOCK profile.

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
BLOCK_PROFILE="$PROFILES_DIR/write-block-${BRANCH_NAME//\//-}-${COMMIT_HASH}-novel.prof"

echo "=========================================================="
echo " Starting End-to-End API Server WRITE BLOCK Contention Benchmark"
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

# We will capture Block profiles for 30 seconds.
curl -s "http://localhost:8001/debug/pprof/block?seconds=30" > "$BLOCK_PROFILE" &

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
    random-label: \"val-\$RAND2-\$RAND1\"
data:
  key1: value1
JSON
    done &
  done
  wait
" || true

echo "   Profiling complete."

echo "=========================================================="
echo "=> Top Block Delays (flat):"
go tool pprof -top -delay "$BLOCK_PROFILE" | head -n 15 || echo "   (No profile data found)"
echo "=========================================================="
echo " Benchmark Complete. Profiles saved. Cluster will now be deleted."
