#!/usr/bin/env bash
# run-brutal-write-contention.sh
# A high-performance contention benchmark designed to force unique.Make() into its slow-path global lock.

set -euo pipefail

CLUSTER_NAME="brutal-bench-cluster"
IMAGE_NAME="brutal-bench-node:latest"
CONCURRENCY=${1:-100}
DURATION="30s"

# Create output directories
PROFILES_DIR="$(pwd)/hack/benchmark/profiles"
mkdir -p "$PROFILES_DIR"

BRANCH_NAME=$(git rev-parse --abbrev-ref HEAD)
COMMIT_HASH=$(git rev-parse --short HEAD)
MUTEX_PROFILE="$PROFILES_DIR/brutal-mutex-${BRANCH_NAME//\//-}-${COMMIT_HASH}.prof"
CPU_PROFILE="$PROFILES_DIR/brutal-cpu-${BRANCH_NAME//\//-}-${COMMIT_HASH}.prof"

echo "=========================================================="
echo " Starting BRUTAL API Server WRITE Contention Benchmark"
echo " Branch: $BRANCH_NAME"
echo " Commit: $COMMIT_HASH"
echo " Concurrency: $CONCURRENCY parallel Go workers"
echo " Duration: $DURATION"
echo "=========================================================="

echo "=> 1. Building Kubernetes Node Image..."
kind build node-image --image "$IMAGE_NAME"

echo "=> 2. Creating Kind Cluster with tuned API Server config..."
kind delete cluster --name "$CLUSTER_NAME" 2>/dev/null || true
# We use the tuned kind.yaml which has high QPS and enables contention profiling
kind create cluster --name "$CLUSTER_NAME" --image "$IMAGE_NAME" --config "$(pwd)/hack/benchmark/kind.yaml"

echo "=> 3. Extracting connection details..."
API_SERVER=$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')
kubectl create clusterrolebinding default-admin --clusterrole=cluster-admin --serviceaccount=default:default > /dev/null 2>&1 || true
TOKEN=$(kubectl create token default --duration=1h)
trap "kind delete cluster --name $CLUSTER_NAME 2>/dev/null || true" EXIT

echo "=> 4. Compiling Brutal Load Generator..."
go build -o /tmp/brutal_client hack/benchmark/brutal_write_client.go

echo "=> 5. Preparing target ConfigMap..."
kubectl create configmap brutal-cm --from-literal=init=true

echo "=> 6. Initiating BRUTAL load and capturing profiles..."

# Capture CPU and Mutex profiles
curl -k -s -H "Authorization: Bearer $TOKEN" "$API_SERVER/debug/pprof/profile?seconds=30" > "$CPU_PROFILE" &
curl -k -s -H "Authorization: Bearer $TOKEN" "$API_SERVER/debug/pprof/mutex?seconds=30" > "$MUTEX_PROFILE" &

# Run the high-performance Go load generator
/tmp/brutal_client -target "$API_SERVER" -token "$TOKEN" -concurrency "$CONCURRENCY" -duration "$DURATION"

echo "   Load complete. Waiting for profiles to finish saving..."
sleep 2

echo "=========================================================="
echo "=> Top Mutex Contention Events:"
go tool pprof -top "$MUTEX_PROFILE" | head -n 15 || echo "   (No contention found)"
echo "=========================================================="
echo "=> Top CPU Hotspots:"
go tool pprof -top "$CPU_PROFILE" | head -n 15 || echo "   (No profile data found)"
echo "=========================================================="
echo " Benchmark Complete. Profiles saved."
