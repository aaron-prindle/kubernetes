#!/usr/bin/env bash
# run-kind-extreme-write-contention.sh
# End-to-end contention benchmark designed to break the spin-phase 
# and force unique.Make to register block delays.

set -euo pipefail

CLUSTER_NAME="contention-bench-cluster"
IMAGE_NAME="contention-bench-node:latest"

# Extreme concurrency to overwhelm the OS scheduler and force goroutine parking.
# We also use lightweight ConfigMaps so the JSON/Auth overhead per request is minimal,
# allowing more concurrent requests to reach unique.Make simultaneously.
CONCURRENCY=500 

# Create output directories
PROFILES_DIR="$(pwd)/hack/benchmark/profiles"
mkdir -p "$PROFILES_DIR"

BRANCH_NAME=$(git rev-parse --abbrev-ref HEAD)
COMMIT_HASH=$(git rev-parse --short HEAD)
BLOCK_PROFILE="$PROFILES_DIR/extreme-block-${BRANCH_NAME//\//-}-${COMMIT_HASH}.prof"
CPU_PROFILE="$PROFILES_DIR/extreme-cpu-${BRANCH_NAME//\//-}-${COMMIT_HASH}.prof"

echo "=========================================================="
echo " Starting API Server EXTREME WRITE Contention Benchmark"
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

echo "=> 4. Initiating EXTREME parallel WRITE requests with NOVEL strings..."

# Capture Block profiles for 30 seconds.
curl -s "http://localhost:8001/debug/pprof/profile?seconds=30" > "$CPU_PROFILE" &
curl -s "http://localhost:8001/debug/pprof/block?seconds=30" > "$BLOCK_PROFILE" &

# Run highly parallel WRITE requests for 30 seconds
# Using 500 parallel bash loops to fire curl as fast as possible.
# Using randomly generated strings in every single payload to force novel unique.Make() calls.
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
    rl1: \"val-\$RAND2-\$RAND1\"
    rl2: \"val-\$RAND1-\$RAND2\"
    rl3: \"val-\$RAND1-\$RAND1\"
    rl4: \"val-\$RAND2-\$RAND2\"
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
go tool pprof -top -total_delay "$BLOCK_PROFILE" | head -n 15 || echo "   (No profile data found)"
echo "=========================================================="
echo " Benchmark Complete. Profiles saved. Cluster will now be deleted."
