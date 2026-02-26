#!/usr/bin/env bash
# run-kind-benchmark.sh
# End-to-end memory benchmark using a live Kubernetes cluster (via kind).
# It builds a node image from the current git tree, spins up a cluster,
# installs Kwok to fake nodes, generates heavy watch cache load using
# Running pods, and captures a heap profile.

set -euo pipefail

CLUSTER_NAME="memory-bench-cluster"
IMAGE_NAME="memory-bench-node:latest"
REPLICAS=${1:-5000}

# Create output directories
PROFILES_DIR="$(pwd)/hack/benchmark/profiles"
mkdir -p "$PROFILES_DIR"

BRANCH_NAME=$(git rev-parse --abbrev-ref HEAD)
COMMIT_HASH=$(git rev-parse --short HEAD)
PROFILE_FILE="$PROFILES_DIR/heap-${BRANCH_NAME//\//-}-${COMMIT_HASH}.prof"

echo "=========================================================="
echo " Starting End-to-End API Server Memory Benchmark (with Kwok)"
echo " Branch: $BRANCH_NAME"
echo " Commit: $COMMIT_HASH"
echo " Target Load: $REPLICAS Running Pods"
echo "=========================================================="

echo "=> 1. Building Kubernetes Node Image from current tree..."
echo "      (This compiles the kube-apiserver with the current branch's changes)"
kind build node-image --image "$IMAGE_NAME"

echo "=> 2. Creating Kind Cluster..."
kind delete cluster --name "$CLUSTER_NAME" 2>/dev/null || true
kind create cluster --name "$CLUSTER_NAME" --image "$IMAGE_NAME"

echo "=> 3. Setting up proxy to API Server..."
kubectl proxy --port=8001 &
PROXY_PID=$!
trap "kill $PROXY_PID 2>/dev/null || true; kind delete cluster --name $CLUSTER_NAME 2>/dev/null || true" EXIT
sleep 2

echo "=> 4. Installing Kwok Controller..."
kubectl apply -f https://github.com/kubernetes-sigs/kwok/releases/download/v0.6.0/kwok.yaml
kubectl apply -f https://github.com/kubernetes-sigs/kwok/releases/download/v0.6.0/stage-fast.yaml
echo "   Waiting for Kwok Controller to be ready..."
sleep 5
kubectl -n kube-system wait --for=condition=Ready pods -l app=kwok-controller --timeout=120s

echo "=> 5. Creating Fake Node..."
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Node
metadata:
  annotations:
    node.alpha.kubernetes.io/ttl: "0"
    kwok.x-k8s.io/node: "fake"
  labels:
    beta.kubernetes.io/arch: amd64
    beta.kubernetes.io/os: linux
    kubernetes.io/arch: amd64
    kubernetes.io/hostname: kwok-node-0
    kubernetes.io/os: linux
    kubernetes.io/role: agent
    node-role.kubernetes.io/agent: ""
    type: kwok
  name: kwok-node-0
spec:
  taints: # We want pods to schedule here normally
status:
  allocatable:
    cpu: 3200
    memory: 25600Gi
    pods: 110000
  capacity:
    cpu: 3200
    memory: 25600Gi
    pods: 110000
  nodeInfo:
    architecture: amd64
    bootID: ""
    containerRuntimeVersion: ""
    kernelVersion: ""
    kubeProxyVersion: ""
    kubeletVersion: fake
    machineID: ""
    operatingSystem: linux
    osImage: ""
    systemUUID: ""
  phase: Running
EOF

echo "=> 6. Deploying load generator (Deployment with $REPLICAS Pods)..."
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: memory-load-gen
spec:
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
EOF

echo "=> 7. Waiting for ReplicaSet to create $REPLICAS pods..."
while true; do
  CREATED=$(kubectl get pods -l app=memory-load-gen --no-headers 2>/dev/null | wc -l || echo 0)
  RUNNING=$(kubectl get pods -l app=memory-load-gen --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l || echo 0)
  if [ "$RUNNING" -ge "$REPLICAS" ]; then
    break
  fi
  echo "   Created $CREATED / $REPLICAS pods ($RUNNING Running)..."
  sleep 5
done

echo "=> 8. All pods Running. Waiting 30 seconds for watch caches to stabilize..."
sleep 30

echo "=> 9. Capturing API Server Heap Profile..."
curl -s http://localhost:8001/debug/pprof/heap > "$PROFILE_FILE"
echo "   Saved heap profile to $PROFILE_FILE"

echo "=========================================================="
echo "=> Top Memory Allocators for FieldsV1 (inuse_space):"
go tool pprof -top -inuse_space "$PROFILE_FILE" | grep -i "FieldsV1" || echo "   (No significant FieldsV1 allocations found)"
echo "=========================================================="
echo " Benchmark Complete. Profile saved. Cluster will now be deleted."
