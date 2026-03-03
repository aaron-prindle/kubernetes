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
kind create cluster --name "$CLUSTER_NAME" --image "$IMAGE_NAME" --config "$(pwd)/hack/benchmark/kind.yaml"

echo "=> 3. Extracting API Server connection details..."
API_SERVER=$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')
kubectl create clusterrolebinding default-admin --clusterrole=cluster-admin --serviceaccount=default:default > /dev/null 2>&1 || true
TOKEN=$(kubectl create token default)
trap "kind delete cluster --name $CLUSTER_NAME 2>/dev/null || true" EXIT
sleep 2

echo "=> 4. Installing Kwok Controller..."
kubectl apply -f https://github.com/kubernetes-sigs/kwok/releases/download/v0.6.0/kwok.yaml
kubectl apply -f https://github.com/kubernetes-sigs/kwok/releases/download/v0.6.0/stage-fast.yaml
echo "   Waiting for Kwok Controller to be ready..."
sleep 30
kubectl -n kube-system wait --for=condition=Ready pods -l app=kwok-controller --timeout=300s

echo "=> 6. Creating 100 Fake Nodes..."
cat << 'EOF' > /tmp/addnodes.sh
#!/bin/bash
PARALLEL_JOBS=100
NODE_COUNT=$1

apply_node() {
  local i=$1
  kubectl apply -f - <<YAML
apiVersion: v1
kind: Node
metadata:
  annotations:
    node.alpha.kubernetes.io/ttl: "5m"
    kwok.x-k8s.io/node: fake
  labels:
    beta.kubernetes.io/arch: amd64
    beta.kubernetes.io/os: linux
    kubernetes.io/arch: amd64
    kubernetes.io/hostname: kwok-node-0
    kubernetes.io/os: linux
    kubernetes.io/role: agent
    node-role.kubernetes.io/agent: ""
    type: kwok
  name: kwok-node-$i
spec:
  taints: # Avoid scheduling actual running pods to fake Node
  - effect: NoSchedule
    key: kwok.x-k8s.io/node
    value: fake
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
    kubeProxyVersion: fake
    kubeletVersion: fake
    machineID: ""
    operatingSystem: linux
    osImage: ""
    systemUUID: ""
  phase: Running
YAML
}

export -f apply_node
seq 1 $NODE_COUNT | xargs -I {} -P $PARALLEL_JOBS bash -c 'apply_node "$@"' _ {}
EOF
chmod +x /tmp/addnodes.sh
/tmp/addnodes.sh 100

echo "=> 7. Deploying load generator (StatefulSet with $REPLICAS Pods)..."
cat <<EOF | kubectl apply -f -
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
        tier: backend
        environment: production
        region: us-east1
        team: infrastructure
        component: super-heavy-processor
        extra-label-1: "value-1"
        extra-label-2: "value-2"
        extra-label-3: "value-3"
        extra-label-4: "value-4"
        extra-label-5: "value-5"
        extra-label-6: "value-6"
        extra-label-7: "value-7"
        extra-label-8: "value-8"
        extra-label-9: "value-9"
        extra-label-10: "value-10"
        extra-label-11: "value-11"
        extra-label-12: "value-12"
        extra-label-13: "value-13"
        extra-label-14: "value-14"
        extra-label-15: "value-15"
        extra-label-16: "value-16"
        extra-label-17: "value-17"
        extra-label-18: "value-18"
        extra-label-19: "value-19"
        extra-label-20: "value-20"
        extra-label-21: "value-21"
        extra-label-22: "value-22"
        extra-label-23: "value-23"
        extra-label-24: "value-24"
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9090"
        security.custom.io/policy: "strict-mode"
        backup.custom.io/schedule: "daily-midnight"
        tracing.custom.io/enabled: "true"
        sidecar.istio.io/inject: "true"
        custom.annotation.1: "value1"
        custom.annotation.2: "value2"
        custom.annotation.3: "value3"
        custom.annotation.4: "value4"
        custom.annotation.5: "value5"
        custom.annotation.6: "value6"
        custom.annotation.7: "value7"
        custom.annotation.8: "value8"
        custom.annotation.9: "value9"
        custom.annotation.10: "value10"
        custom.annotation.11: "value11"
        custom.annotation.12: "value12"
        custom.annotation.13: "value13"
        custom.annotation.14: "value14"
        custom.annotation.15: "value15"
        custom.annotation.16: "value16"
        custom.annotation.17: "value17"
        custom.annotation.18: "value18"
        custom.annotation.19: "value19"
        custom.annotation.20: "value20"
        custom.annotation.21: "value21"
        custom.annotation.22: "value22"
        custom.annotation.23: "value23"
        custom.annotation.24: "value24"
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
      initContainers:
      - name: init-config
        image: registry.k8s.io/pause:3.9
        env:
        - name: INIT_VAR_1
          value: "config_startup_sequence_001"
        - name: INIT_VAR_2
          value: "config_startup_sequence_002"
        - name: INIT_VAR_3
          value: "config_startup_sequence_003"
        - name: INIT_VAR_4
          value: "config_startup_sequence_004"
        - name: INIT_VAR_5
          value: "config_startup_sequence_005"
        - name: INIT_VAR_6
          value: "config_startup_sequence_006"
        - name: INIT_VAR_7
          value: "config_startup_sequence_007"
        - name: INIT_VAR_8
          value: "config_startup_sequence_008"
        - name: INIT_VAR_9
          value: "config_startup_sequence_009"
        - name: INIT_VAR_10
          value: "config_startup_sequence_010"
        - name: INIT_VAR_11
          value: "config_startup_sequence_011"
        - name: INIT_VAR_12
          value: "config_startup_sequence_012"
        - name: INIT_VAR_13
          value: "config_startup_sequence_013"
        - name: INIT_VAR_14
          value: "config_startup_sequence_014"
        - name: INIT_VAR_15
          value: "config_startup_sequence_015"
        - name: INIT_VAR_16
          value: "config_startup_sequence_016"
        - name: INIT_VAR_17
          value: "config_startup_sequence_017"
        - name: INIT_VAR_18
          value: "config_startup_sequence_018"
        - name: INIT_VAR_19
          value: "config_startup_sequence_019"
        - name: INIT_VAR_20
          value: "config_startup_sequence_020"
      - name: init-network
        image: registry.k8s.io/pause:3.9
        env:
        - name: NET_VAR_1
          value: "network_startup_sequence_001"
        - name: NET_VAR_2
          value: "network_startup_sequence_002"
        - name: NET_VAR_3
          value: "network_startup_sequence_003"
        - name: NET_VAR_4
          value: "network_startup_sequence_004"
        - name: NET_VAR_5
          value: "network_startup_sequence_005"
        - name: NET_VAR_6
          value: "network_startup_sequence_006"
        - name: NET_VAR_7
          value: "network_startup_sequence_007"
        - name: NET_VAR_8
          value: "network_startup_sequence_008"
        - name: NET_VAR_9
          value: "network_startup_sequence_009"
        - name: NET_VAR_10
          value: "network_startup_sequence_010"
        - name: NET_VAR_11
          value: "network_startup_sequence_011"
        - name: NET_VAR_12
          value: "network_startup_sequence_012"
      containers:
      - name: heavy-worker-1
        image: registry.k8s.io/pause:3.9
        env:
        - name: WORKER_CONFIG_1
          value: "very_long_configuration_string_1"
        - name: WORKER_CONFIG_2
          value: "very_long_configuration_string_2"
        - name: WORKER_CONFIG_3
          value: "very_long_configuration_string_3"
        - name: WORKER_CONFIG_4
          value: "very_long_configuration_string_4"
        - name: WORKER_CONFIG_5
          value: "very_long_configuration_string_5"
        - name: WORKER_CONFIG_6
          value: "very_long_configuration_string_6"
        - name: WORKER_CONFIG_7
          value: "very_long_configuration_string_7"
        - name: WORKER_CONFIG_8
          value: "very_long_configuration_string_8"
        - name: WORKER_CONFIG_9
          value: "very_long_configuration_string_9"
        - name: WORKER_CONFIG_10
          value: "very_long_configuration_string_10"
        - name: WORKER_CONFIG_11
          value: "very_long_configuration_string_11"
        - name: WORKER_CONFIG_12
          value: "very_long_configuration_string_12"
        - name: WORKER_CONFIG_13
          value: "very_long_configuration_string_13"
        - name: WORKER_CONFIG_14
          value: "very_long_configuration_string_14"
        - name: WORKER_CONFIG_15
          value: "very_long_configuration_string_15"
        - name: WORKER_CONFIG_16
          value: "very_long_configuration_string_16"
        - name: WORKER_CONFIG_17
          value: "very_long_configuration_string_17"
        - name: WORKER_CONFIG_18
          value: "very_long_configuration_string_18"
        - name: WORKER_CONFIG_19
          value: "very_long_configuration_string_19"
        - name: WORKER_CONFIG_20
          value: "very_long_configuration_string_20"
        - name: WORKER_CONFIG_21
          value: "very_long_configuration_string_21"
        - name: WORKER_CONFIG_22
          value: "very_long_configuration_string_22"
        - name: WORKER_CONFIG_23
          value: "very_long_configuration_string_23"
        - name: WORKER_CONFIG_24
          value: "very_long_configuration_string_24"
        - name: WORKER_CONFIG_25
          value: "very_long_configuration_string_25"
        - name: WORKER_CONFIG_26
          value: "very_long_configuration_string_26"
        - name: WORKER_CONFIG_27
          value: "very_long_configuration_string_27"
        - name: WORKER_CONFIG_28
          value: "very_long_configuration_string_28"
        - name: WORKER_CONFIG_29
          value: "very_long_configuration_string_29"
        - name: WORKER_CONFIG_30
          value: "very_long_configuration_string_30"
        volumeMounts:
        - name: cache-vol-1
          mountPath: /var/cache/worker1/1
        - name: cache-vol-2
          mountPath: /var/cache/worker1/2
        - name: cache-vol-3
          mountPath: /var/cache/worker1/3
        - name: cache-vol-4
          mountPath: /var/cache/worker1/4
        - name: cache-vol-5
          mountPath: /var/cache/worker1/5
        - name: cache-vol-6
          mountPath: /var/cache/worker1/6
        - name: cache-vol-7
          mountPath: /var/cache/worker1/7
        - name: cache-vol-8
          mountPath: /var/cache/worker1/8
        - name: cache-vol-9
          mountPath: /var/cache/worker1/9
        - name: cache-vol-10
          mountPath: /var/cache/worker1/10
        - name: cache-vol-11
          mountPath: /var/cache/worker1/11
        - name: cache-vol-12
          mountPath: /var/cache/worker1/12
        - name: cache-vol-13
          mountPath: /var/cache/worker1/13
        - name: cache-vol-14
          mountPath: /var/cache/worker1/14
        - name: cache-vol-15
          mountPath: /var/cache/worker1/15
        - name: cache-vol-16
          mountPath: /var/cache/worker1/16
        - name: cache-vol-17
          mountPath: /var/cache/worker1/17
        - name: cache-vol-18
          mountPath: /var/cache/worker1/18
        - name: cache-vol-19
          mountPath: /var/cache/worker1/19
        - name: cache-vol-20
          mountPath: /var/cache/worker1/20
        - name: cache-vol-21
          mountPath: /var/cache/worker1/21
        - name: cache-vol-22
          mountPath: /var/cache/worker1/22
      - name: heavy-worker-2
        image: registry.k8s.io/pause:3.9
        env:
        - name: SECONDARY_CONFIG_1
          value: "secondary_system_parameter_1"
        - name: SECONDARY_CONFIG_2
          value: "secondary_system_parameter_2"
        - name: SECONDARY_CONFIG_3
          value: "secondary_system_parameter_3"
        - name: SECONDARY_CONFIG_4
          value: "secondary_system_parameter_4"
        - name: SECONDARY_CONFIG_5
          value: "secondary_system_parameter_5"
        - name: SECONDARY_CONFIG_6
          value: "secondary_system_parameter_6"
        - name: SECONDARY_CONFIG_7
          value: "secondary_system_parameter_7"
        - name: SECONDARY_CONFIG_8
          value: "secondary_system_parameter_8"
        - name: SECONDARY_CONFIG_9
          value: "secondary_system_parameter_9"
        - name: SECONDARY_CONFIG_10
          value: "secondary_system_parameter_10"
        - name: SECONDARY_CONFIG_11
          value: "secondary_system_parameter_11"
        - name: SECONDARY_CONFIG_12
          value: "secondary_system_parameter_12"
        - name: SECONDARY_CONFIG_13
          value: "secondary_system_parameter_13"
        - name: SECONDARY_CONFIG_14
          value: "secondary_system_parameter_14"
        - name: SECONDARY_CONFIG_15
          value: "secondary_system_parameter_15"
        - name: SECONDARY_CONFIG_16
          value: "secondary_system_parameter_16"
        - name: SECONDARY_CONFIG_17
          value: "secondary_system_parameter_17"
        - name: SECONDARY_CONFIG_18
          value: "secondary_system_parameter_18"
        volumeMounts:
        - name: cache-vol-23
          mountPath: /var/lib/data/23
        - name: cache-vol-24
          mountPath: /var/lib/data/24
        - name: cache-vol-25
          mountPath: /var/lib/data/25
        - name: cache-vol-26
          mountPath: /var/lib/data/26
        - name: cache-vol-27
          mountPath: /var/lib/data/27
        - name: cache-vol-28
          mountPath: /var/lib/data/28
        - name: cache-vol-29
          mountPath: /var/lib/data/29
        - name: cache-vol-30
          mountPath: /var/lib/data/30
        - name: cache-vol-31
          mountPath: /var/lib/data/31
        - name: cache-vol-32
          mountPath: /var/lib/data/32
        - name: cache-vol-33
          mountPath: /var/lib/data/33
        - name: cache-vol-34
          mountPath: /var/lib/data/34
        - name: cache-vol-35
          mountPath: /var/lib/data/35
        - name: cache-vol-36
          mountPath: /var/lib/data/36
        - name: cache-vol-37
          mountPath: /var/lib/data/37
        - name: cache-vol-38
          mountPath: /var/lib/data/38
        - name: cache-vol-39
          mountPath: /var/lib/data/39
        - name: cache-vol-40
          mountPath: /var/lib/data/40
      - name: sidecar-logger
        image: registry.k8s.io/pause:3.9
        env:
        - name: LOG_VAR_1
          value: "log_config_value_1"
        - name: LOG_VAR_2
          value: "log_config_value_2"
        - name: LOG_VAR_3
          value: "log_config_value_3"
        - name: LOG_VAR_4
          value: "log_config_value_4"
        - name: LOG_VAR_5
          value: "log_config_value_5"
        - name: LOG_VAR_6
          value: "log_config_value_6"
        - name: LOG_VAR_7
          value: "log_config_value_7"
        - name: LOG_VAR_8
          value: "log_config_value_8"
        - name: LOG_VAR_9
          value: "log_config_value_9"
        - name: LOG_VAR_10
          value: "log_config_value_10"
        - name: LOG_VAR_11
          value: "log_config_value_11"
        - name: LOG_VAR_12
          value: "log_config_value_12"
        - name: LOG_VAR_13
          value: "log_config_value_13"
        - name: LOG_VAR_14
          value: "log_config_value_14"
      volumes:
      - name: cache-vol-1
        emptyDir: {}
      - name: cache-vol-2
        emptyDir: {}
      - name: cache-vol-3
        emptyDir: {}
      - name: cache-vol-4
        emptyDir: {}
      - name: cache-vol-5
        emptyDir: {}
      - name: cache-vol-6
        emptyDir: {}
      - name: cache-vol-7
        emptyDir: {}
      - name: cache-vol-8
        emptyDir: {}
      - name: cache-vol-9
        emptyDir: {}
      - name: cache-vol-10
        emptyDir: {}
      - name: cache-vol-11
        emptyDir: {}
      - name: cache-vol-12
        emptyDir: {}
      - name: cache-vol-13
        emptyDir: {}
      - name: cache-vol-14
        emptyDir: {}
      - name: cache-vol-15
        emptyDir: {}
      - name: cache-vol-16
        emptyDir: {}
      - name: cache-vol-17
        emptyDir: {}
      - name: cache-vol-18
        emptyDir: {}
      - name: cache-vol-19
        emptyDir: {}
      - name: cache-vol-20
        emptyDir: {}
      - name: cache-vol-21
        emptyDir: {}
      - name: cache-vol-22
        emptyDir: {}
      - name: cache-vol-23
        emptyDir: {}
      - name: cache-vol-24
        emptyDir: {}
      - name: cache-vol-25
        emptyDir: {}
      - name: cache-vol-26
        emptyDir: {}
      - name: cache-vol-27
        emptyDir: {}
      - name: cache-vol-28
        emptyDir: {}
      - name: cache-vol-29
        emptyDir: {}
      - name: cache-vol-30
        emptyDir: {}
      - name: cache-vol-31
        emptyDir: {}
      - name: cache-vol-32
        emptyDir: {}
      - name: cache-vol-33
        emptyDir: {}
      - name: cache-vol-34
        emptyDir: {}
      - name: cache-vol-35
        emptyDir: {}
      - name: cache-vol-36
        emptyDir: {}
      - name: cache-vol-37
        emptyDir: {}
      - name: cache-vol-38
        emptyDir: {}
      - name: cache-vol-39
        emptyDir: {}
      - name: cache-vol-40
        emptyDir: {}
EOF
echo "=> 8. Waiting for StatefulSet to create $REPLICAS pods..."
while true; do
  RUNNING=$(kubectl get statefulset memory-load-gen -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo 0)
  if [ -z "$RUNNING" ]; then
    RUNNING=0
  fi
  if [ "$RUNNING" -ge "$REPLICAS" ]; then
    break
  fi
  echo "   $RUNNING / $REPLICAS pods Running..."
  sleep 5
done

echo "=> 8. All pods Running. Waiting 30 seconds for watch caches to stabilize..."
sleep 30

echo "=> 9. Capturing API Server Heap Profile..."
curl -k -s -H "Authorization: Bearer $TOKEN" "$API_SERVER/debug/pprof/heap" > "$PROFILE_FILE"
echo "   Saved heap profile to $PROFILE_FILE"

echo "=========================================================="
echo "=> Top Memory Allocators for FieldsV1 (inuse_space):"
go tool pprof -top -inuse_space "$PROFILE_FILE" | grep -i "FieldsV1" || echo "   (No significant FieldsV1 allocations found)"
echo "=========================================================="
echo " Benchmark Complete. Profile saved. Cluster will now be deleted."
