#!/usr/bin/env bash
# run-kind-baseline-scaling-benchmark-100k.sh
# Runs ONLY the 100k memory test against the current branch.

set -euo pipefail

CLUSTER_NAME="baseline-scaling-cluster"
IMAGE_NAME="baseline-scaling-node:latest"

# ONLY test 100k
SCALES=(100000)

# Create output directories
PROFILES_DIR="$(pwd)/hack/benchmark/profiles"
mkdir -p "$PROFILES_DIR"

BRANCH_NAME=$(git rev-parse --abbrev-ref HEAD)
COMMIT_HASH=$(git rev-parse --short HEAD)

echo "=========================================================="
echo " Starting 100k Scaling Benchmark (managedFields Bloat)"
echo " Branch: $BRANCH_NAME"
echo " Commit: $COMMIT_HASH"
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

echo "=> 4. Installing Kwok Controller..."
kubectl apply -f https://github.com/kubernetes-sigs/kwok/releases/download/v0.6.0/kwok.yaml
kubectl apply -f https://github.com/kubernetes-sigs/kwok/releases/download/v0.6.0/stage-fast.yaml
echo "   Waiting for Kwok Controller to be ready..."
sleep 15
kubectl -n kube-system wait --for=condition=Ready pods -l app=kwok-controller --timeout=300s

echo "=> 5. Creating Fake Nodes using addnodes.sh..."
# Pre-provision enough nodes for the max scale (100,000 pods requires ~1000 nodes at 110 pods/node)
"$(pwd)/hack/benchmark/addnodes.sh" 1000

for REPLICAS in "${SCALES[@]}"; do
    echo "=========================================================="
    echo "=> 6. Testing Scale: $REPLICAS Pods"
    
    # Deploy or scale the StatefulSet
    cat <<YAML | kubectl apply -f -
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: memory-load-gen
spec:
  podManagementPolicy: "Parallel"
  replicas: $REPLICAS
  selector:
    matchLabels:
      app: memory-load-gen
  template:
    metadata:
      labels:
        app: memory-load-gen
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
YAML

    echo "=> 7. Waiting for $REPLICAS pods to be Running..."
    while true; do
      RUNNING=$(kubectl get pods -l app=memory-load-gen --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l || echo 0)
      if [ "$RUNNING" -ge "$REPLICAS" ]; then
        break
      fi
      echo "   ($RUNNING / $REPLICAS) Running..."
      sleep 10
    done

    echo "=> 8. Reached $REPLICAS pods. Waiting 30 seconds for WatchCache to stabilize..."
    sleep 30
    
    PROFILE_FILE="$PROFILES_DIR/baseline-scaling-${REPLICAS}-${COMMIT_HASH}.prof"
    echo "=> 9. Capturing API Server Heap Profile for $REPLICAS pods..."
    curl -s http://localhost:8001/debug/pprof/heap > "$PROFILE_FILE"
    
    echo "=> 10. FieldsV1 (inuse_space) for $REPLICAS pods:"
    go tool pprof -top -inuse_space "$PROFILE_FILE" | grep -i "FieldsV1" || echo "   (No significant FieldsV1 allocations found)"
    echo "----------------------------------------------------------"
done

echo "=========================================================="
echo " 100k Scaling Benchmark Complete."