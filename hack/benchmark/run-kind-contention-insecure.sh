#!/usr/bin/env bash
# run-kind-contention-insecure.sh
# End-to-end contention benchmark bypassing TLS and Auth via debug socket.

set -euo pipefail

CLUSTER_NAME="contention-insecure-cluster"
IMAGE_NAME="contention-bench-node:latest"
CONCURRENCY=1000

# Create output directories
PROFILES_DIR="$(pwd)/hack/benchmark/profiles"
mkdir -p "$PROFILES_DIR"

BRANCH_NAME=$(git rev-parse --abbrev-ref HEAD)
COMMIT_HASH=$(git rev-parse --short HEAD)
BLOCK_PROFILE="$PROFILES_DIR/insecure-block-${BRANCH_NAME//\//-}-${COMMIT_HASH}.prof"

echo "=========================================================="
echo " Starting API Server INSECURE SOCKET Contention Benchmark"
echo " Branch: $BRANCH_NAME"
echo " Concurrency: $CONCURRENCY parallel clients"
echo "=========================================================="

echo "=> 1. Creating Kind Config..."
cat << 'KIND' > /tmp/kind-insecure.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: ClusterConfiguration
    apiServer:
      extraArgs:
        max-requests-inflight: "5000"
        max-mutating-requests-inflight: "5000"
        v: "2"
        profiling: "true"
        contention-profiling: "true"
        # Open an unauthenticated unix socket directly to the API handler
        debug-socket-path: "/var/run/kubernetes/apiserver-debug.sock"
        anonymous-auth: "true"
KIND

echo "=> 2. Building Node Image..."
kind build node-image --image "$IMAGE_NAME"

echo "=> 3. Creating Kind Cluster..."
kind delete cluster --name "$CLUSTER_NAME" 2>/dev/null || true
kind create cluster --name "$CLUSTER_NAME" --image "$IMAGE_NAME" --config /tmp/kind-insecure.yaml

echo "=> 4. Setting up Socat proxy to internal Unix socket..."
# Forward a local TCP port to the internal unix socket to bypass the entire TLS/Auth stack
docker exec -d ${CLUSTER_NAME}-control-plane sh -c "apt-get update && apt-get install -y socat && socat TCP-LISTEN:8080,fork,reuseaddr UNIX-CONNECT:/var/run/kubernetes/apiserver-debug.sock"
sleep 10

# Expose that container port to our host
kubectl port-forward -n kube-system pod/kube-apiserver-${CLUSTER_NAME}-control-plane 8080:8080 &
PROXY_PID=$!
trap "kill $PROXY_PID 2>/dev/null || true; kind delete cluster --name $CLUSTER_NAME 2>/dev/null || true" EXIT
sleep 5

echo "=> 5. Initiating INSECURE parallel WRITE requests..."
curl -s "http://localhost:8080/debug/pprof/block?seconds=20" > "$BLOCK_PROFILE" &

timeout 22s bash -c "
  for i in \$(seq 1 $CONCURRENCY); do
    while true; do
      RAND1=\$RANDOM
      cat <<JSON | curl -m 2 -s -X POST -H 'Content-Type: application/json' http://localhost:8080/api/v1/namespaces/default/configmaps > /dev/null || true
{
  \"apiVersion\": \"v1\",
  \"kind\": \"ConfigMap\",
  \"metadata\": {
    \"name\": \"cm-\$i-\$RAND1\",
    \"labels\": { \"rl1\": \"val-\$RAND1\" }
  }
}
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
