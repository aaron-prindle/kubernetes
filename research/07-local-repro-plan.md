# Local Reproduction Plan: SSA ManagedFields Memory Bottleneck

## Objective
Create a reproducible local environment that demonstrates the apiserver memory impact of managedFields at scale, and provides a testing framework for evaluating solutions.

## Prerequisites
- Docker Desktop or equivalent container runtime
- `kind` (Kubernetes in Docker) installed
- `kubectl` installed
- `go` toolchain (1.22+) for building custom apiserver
- `pprof` for memory profiling
- ~16 GB RAM on host machine

## Phase 1: Cluster Setup

### 1.1 Kind Cluster Configuration

```yaml
# kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: ClusterConfiguration
    apiServer:
      extraArgs:
        # Enable profiling endpoint
        profiling: "true"
        # Increase event cache for testing
        default-watch-cache-size: "5000"
        # Enable verbose logging for memory tracking
        v: "4"
  extraPortMappings:
  - containerPort: 6443
    hostPort: 6443
  - containerPort: 8001
    hostPort: 8001
```

```bash
# Create the cluster
kind create cluster --config kind-config.yaml --name ssa-memory-test
```

### 1.2 Enable pprof Access

```bash
# Port-forward to apiserver pprof endpoint
kubectl proxy --port=8001 &

# Test pprof access
curl http://localhost:8001/debug/pprof/heap > /dev/null
```

## Phase 2: Baseline Measurement

### 2.1 Measure Empty Cluster Memory

```bash
#!/bin/bash
# baseline-measurement.sh

echo "=== Baseline Memory Measurement ==="

# Get apiserver container ID
APISERVER_CONTAINER=$(docker ps --filter "name=ssa-memory-test-control-plane" -q)

# Record RSS
docker stats --no-stream $APISERVER_CONTAINER

# Take heap profile
curl -s http://localhost:8001/debug/pprof/heap > profiles/baseline-heap.prof

# Count objects
kubectl get pods -A --no-headers | wc -l
kubectl get configmaps -A --no-headers | wc -l
kubectl get secrets -A --no-headers | wc -l

echo "Baseline recorded."
```

### 2.2 Record Initial Metrics

```bash
mkdir -p profiles measurements

# Heap profile
curl -s http://localhost:8001/debug/pprof/heap > profiles/baseline-heap.prof

# Allocs profile
curl -s http://localhost:8001/debug/pprof/allocs > profiles/baseline-allocs.prof

# Goroutines
curl -s http://localhost:8001/debug/pprof/goroutine > profiles/baseline-goroutine.prof

# Memory stats via pprof
go tool pprof -text profiles/baseline-heap.prof | head -30 > measurements/baseline-top-allocators.txt
```

## Phase 3: Generate Load with SSA Objects

### 3.1 Create Objects Using Server-Side Apply

```bash
#!/bin/bash
# generate-ssa-objects.sh
# Creates N objects using SSA with M different managers

NUM_OBJECTS=${1:-5000}
NUM_MANAGERS=${2:-5}
NAMESPACE="ssa-test"

kubectl create namespace $NAMESPACE 2>/dev/null

echo "Creating $NUM_OBJECTS ConfigMaps with $NUM_MANAGERS managers each..."

for i in $(seq 1 $NUM_OBJECTS); do
    # Create ConfigMap with several data entries to make managedFields substantial
    for m in $(seq 1 $NUM_MANAGERS); do
        cat <<EOF | kubectl apply --server-side --field-manager="manager-$m" -f - 2>/dev/null
apiVersion: v1
kind: ConfigMap
metadata:
  name: cm-$i
  namespace: $NAMESPACE
data:
  key-m${m}-1: "value-${i}-${m}-1"
  key-m${m}-2: "value-${i}-${m}-2"
  key-m${m}-3: "value-${i}-${m}-3"
  key-m${m}-4: "value-${i}-${m}-4"
  key-m${m}-5: "value-${i}-${m}-5"
EOF
    done

    if [ $((i % 100)) -eq 0 ]; then
        echo "Created $i / $NUM_OBJECTS objects"
    fi
done

echo "Done creating objects."
```

### 3.2 Create Complex Objects (Deployments with SSA)

```bash
#!/bin/bash
# generate-ssa-deployments.sh
# Creates Deployments with complex specs for larger managedFields

NUM_DEPLOYMENTS=${1:-1000}
NUM_MANAGERS=${2:-3}
NAMESPACE="ssa-deploy-test"

kubectl create namespace $NAMESPACE 2>/dev/null

for i in $(seq 1 $NUM_DEPLOYMENTS); do
    for m in $(seq 1 $NUM_MANAGERS); do
        cat <<EOF | kubectl apply --server-side --field-manager="deploy-manager-$m" -f - 2>/dev/null
apiVersion: apps/v1
kind: Deployment
metadata:
  name: deploy-$i
  namespace: $NAMESPACE
  labels:
    app: test-$i
    tier: backend
    version: v1
    manager: "manager-$m"
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test-$i
  template:
    metadata:
      labels:
        app: test-$i
        tier: backend
    spec:
      containers:
      - name: container-1
        image: nginx:latest
        ports:
        - containerPort: 80
        env:
        - name: ENV_VAR_1
          value: "value1"
        - name: ENV_VAR_2
          value: "value2"
        - name: ENV_VAR_3
          value: "value3"
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 200m
            memory: 256Mi
        volumeMounts:
        - name: data
          mountPath: /data
      volumes:
      - name: data
        emptyDir: {}
EOF
    done

    if [ $((i % 100)) -eq 0 ]; then
        echo "Created $i / $NUM_DEPLOYMENTS deployments"
    fi
done
```

### 3.3 Bulk Creation with Go Client (Faster)

```go
// cmd/ssa-load-generator/main.go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "sync"
    "time"

    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
    "k8s.io/apimachinery/pkg/types"
    "k8s.io/client-go/dynamic"
    "k8s.io/client-go/tools/clientcmd"
    "k8s.io/apimachinery/pkg/runtime/schema"
)

var (
    numObjects  = flag.Int("objects", 5000, "number of objects to create")
    numManagers = flag.Int("managers", 5, "number of managers per object")
    concurrency = flag.Int("concurrency", 10, "concurrent workers")
    namespace   = flag.String("namespace", "ssa-test", "target namespace")
)

func main() {
    flag.Parse()

    config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
    if err != nil {
        log.Fatal(err)
    }

    client, err := dynamic.NewForConfig(config)
    if err != nil {
        log.Fatal(err)
    }

    gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}

    sem := make(chan struct{}, *concurrency)
    var wg sync.WaitGroup

    start := time.Now()

    for i := 0; i < *numObjects; i++ {
        wg.Add(1)
        sem <- struct{}{}
        go func(idx int) {
            defer wg.Done()
            defer func() { <-sem }()

            for m := 0; m < *numManagers; m++ {
                obj := &unstructured.Unstructured{
                    Object: map[string]interface{}{
                        "apiVersion": "v1",
                        "kind":       "ConfigMap",
                        "metadata": map[string]interface{}{
                            "name":      fmt.Sprintf("cm-%d", idx),
                            "namespace": *namespace,
                        },
                        "data": map[string]interface{}{
                            fmt.Sprintf("key-m%d-1", m): fmt.Sprintf("val-%d-%d-1", idx, m),
                            fmt.Sprintf("key-m%d-2", m): fmt.Sprintf("val-%d-%d-2", idx, m),
                            fmt.Sprintf("key-m%d-3", m): fmt.Sprintf("val-%d-%d-3", idx, m),
                            fmt.Sprintf("key-m%d-4", m): fmt.Sprintf("val-%d-%d-4", idx, m),
                            fmt.Sprintf("key-m%d-5", m): fmt.Sprintf("val-%d-%d-5", idx, m),
                        },
                    },
                }

                _, err := client.Resource(gvr).Namespace(*namespace).Apply(
                    context.TODO(),
                    fmt.Sprintf("cm-%d", idx),
                    obj,
                    metav1.ApplyOptions{
                        FieldManager: fmt.Sprintf("manager-%d", m),
                        Force:        true,
                    },
                )
                if err != nil {
                    log.Printf("Error applying cm-%d with manager-%d: %v", idx, m, err)
                }
            }

            if idx%500 == 0 {
                log.Printf("Progress: %d/%d objects", idx, *numObjects)
            }
        }(i)
    }

    wg.Wait()
    elapsed := time.Since(start)
    log.Printf("Created %d objects with %d managers each in %v", *numObjects, *numManagers, elapsed)
}
```

## Phase 4: Measurement and Analysis

### 4.1 Post-Load Memory Measurement

```bash
#!/bin/bash
# measure-after-load.sh

echo "=== Post-Load Memory Measurement ==="

# Docker stats
docker stats --no-stream $(docker ps --filter "name=ssa-memory-test-control-plane" -q)

# Heap profile
curl -s http://localhost:8001/debug/pprof/heap > profiles/postload-heap.prof

# Compare with baseline
echo "=== Memory Delta ==="
go tool pprof -text profiles/postload-heap.prof | head -40 > measurements/postload-top-allocators.txt

# Diff
diff measurements/baseline-top-allocators.txt measurements/postload-top-allocators.txt
```

### 4.2 Analyze ManagedFields Contribution

```bash
#!/bin/bash
# analyze-managed-fields.sh

# Get total size of managedFields across all objects in a namespace
kubectl get configmaps -n ssa-test -o json | \
    python3 -c "
import json, sys
data = json.load(sys.stdin)
total_mf_size = 0
total_obj_size = 0
for item in data['items']:
    obj_json = json.dumps(item)
    total_obj_size += len(obj_json)
    mf = item.get('metadata', {}).get('managedFields', [])
    mf_json = json.dumps(mf)
    total_mf_size += len(mf_json)

print(f'Total objects: {len(data[\"items\"])}')
print(f'Total object size: {total_obj_size / 1024 / 1024:.2f} MB')
print(f'Total managedFields size: {total_mf_size / 1024 / 1024:.2f} MB')
print(f'managedFields as % of total: {total_mf_size / total_obj_size * 100:.1f}%')
print(f'Average managedFields per object: {total_mf_size / len(data[\"items\"]) / 1024:.2f} KB')
"
```

### 4.3 Heap Profile Analysis

```bash
# Analyze heap profile for managedFields-related allocations
go tool pprof -http=:9090 profiles/postload-heap.prof

# Look for these allocation sites:
# - k8s.io/apimachinery/pkg/apis/meta/v1.(*FieldsV1).UnmarshalJSON
# - k8s.io/apimachinery/pkg/util/managedfields/internal.FieldsToSet
# - sigs.k8s.io/structured-merge-diff/v6/fieldpath.(*Set).FromJSON
# - runtime.Object.DeepCopyObject (includes managedFields copy)
```

### 4.4 Compare With and Without ManagedFields

```bash
#!/bin/bash
# compare-with-without-mf.sh

# Measure object sizes with managedFields
kubectl get configmaps -n ssa-test -o json | wc -c > measurements/with-mf-size.txt

# Measure without (strip managedFields)
kubectl get configmaps -n ssa-test -o json | \
    python3 -c "
import json, sys
data = json.load(sys.stdin)
for item in data['items']:
    item['metadata'].pop('managedFields', None)
print(json.dumps(data))
" | wc -c > measurements/without-mf-size.txt

echo "With managedFields: $(cat measurements/with-mf-size.txt) bytes"
echo "Without managedFields: $(cat measurements/without-mf-size.txt) bytes"
```

## Phase 5: Test Solutions

### 5.1 Test Compression Approach

```go
// cmd/compression-test/main.go
// Test compression ratios on actual managedFields data

package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    "github.com/klauspost/compress/zstd"
    "github.com/golang/snappy"

    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
)

func main() {
    config, _ := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
    clientset, _ := kubernetes.NewForConfig(config)

    cms, _ := clientset.CoreV1().ConfigMaps("ssa-test").List(
        context.TODO(), metav1.ListOptions{Limit: 1000})

    totalRaw := 0
    totalSnappy := 0
    totalZstd := 0

    encoder, _ := zstd.NewWriter(nil)

    for _, cm := range cms.Items {
        for _, mf := range cm.ManagedFields {
            if mf.FieldsV1 != nil {
                raw := mf.FieldsV1.Raw
                totalRaw += len(raw)

                snappyData := snappy.Encode(nil, raw)
                totalSnappy += len(snappyData)

                zstdData := encoder.EncodeAll(raw, nil)
                totalZstd += len(zstdData)
            }
        }
    }

    fmt.Printf("Objects analyzed: %d\n", len(cms.Items))
    fmt.Printf("Total FieldsV1 raw: %d bytes (%.2f MB)\n", totalRaw, float64(totalRaw)/1024/1024)
    fmt.Printf("Total Snappy: %d bytes (%.2f MB, %.1f%% savings)\n",
        totalSnappy, float64(totalSnappy)/1024/1024, (1-float64(totalSnappy)/float64(totalRaw))*100)
    fmt.Printf("Total Zstd: %d bytes (%.2f MB, %.1f%% savings)\n",
        totalZstd, float64(totalZstd)/1024/1024, (1-float64(totalZstd)/float64(totalRaw))*100)
}
```

### 5.2 Test Stripping Impact

```bash
#!/bin/bash
# test-strip-impact.sh
# Simulates the effect of stripping managedFields on memory

echo "=== Stripping Impact Simulation ==="

# Measure current apiserver memory
BEFORE_RSS=$(docker exec ssa-memory-test-control-plane cat /proc/$(docker exec ssa-memory-test-control-plane pidof kube-apiserver)/status | grep VmRSS | awk '{print $2}')
echo "Before RSS: ${BEFORE_RSS} kB"

# Calculate theoretical savings
python3 -c "
rss_kb = $BEFORE_RSS
# Estimate managedFields contribution (from Phase 4 analysis)
# Use actual measurements from Phase 4
mf_contribution_pct = 0.25  # Placeholder - use actual measurement
savings_kb = rss_kb * mf_contribution_pct
print(f'Current RSS: {rss_kb} kB ({rss_kb/1024:.0f} MB)')
print(f'Estimated managedFields contribution: {savings_kb:.0f} kB ({savings_kb/1024:.0f} MB)')
print(f'Projected RSS after stripping: {rss_kb - savings_kb:.0f} kB ({(rss_kb - savings_kb)/1024:.0f} MB)')
print(f'Projected savings: {mf_contribution_pct*100:.0f}%')
"
```

## Phase 6: Results Documentation

### 6.1 Create Results Report

```bash
#!/bin/bash
# collect-results.sh

echo "# SSA Memory Bottleneck - Local Reproduction Results" > measurements/results.md
echo "" >> measurements/results.md
echo "## Environment" >> measurements/results.md
echo "- Kind cluster: $(kind version)" >> measurements/results.md
echo "- Kubernetes: $(kubectl version --short 2>/dev/null)" >> measurements/results.md
echo "- Host RAM: $(sysctl -n hw.memsize 2>/dev/null || free -h | grep Mem | awk '{print $2}')" >> measurements/results.md
echo "" >> measurements/results.md

echo "## Object Counts" >> measurements/results.md
echo "- ConfigMaps (ssa-test): $(kubectl get cm -n ssa-test --no-headers 2>/dev/null | wc -l)" >> measurements/results.md
echo "- Deployments (ssa-deploy-test): $(kubectl get deploy -n ssa-deploy-test --no-headers 2>/dev/null | wc -l)" >> measurements/results.md
echo "" >> measurements/results.md

echo "## Memory Measurements" >> measurements/results.md
echo "See profiles/ directory for pprof data" >> measurements/results.md
echo "" >> measurements/results.md

cat measurements/results.md
```

## Automation Script

```bash
#!/bin/bash
# run-full-repro.sh
# Runs the full reproduction and measurement pipeline

set -e

echo "=== SSA Memory Bottleneck Reproduction ==="
echo "Starting at $(date)"

# Phase 1: Setup
echo "Phase 1: Creating cluster..."
kind create cluster --config kind-config.yaml --name ssa-memory-test 2>/dev/null || true
kubectl proxy --port=8001 &
PROXY_PID=$!
sleep 5

# Phase 2: Baseline
echo "Phase 2: Baseline measurement..."
mkdir -p profiles measurements
bash baseline-measurement.sh

# Phase 3: Generate load
echo "Phase 3: Generating SSA objects..."
kubectl create namespace ssa-test 2>/dev/null || true
bash generate-ssa-objects.sh 5000 5
sleep 30  # Wait for cache to stabilize

# Phase 4: Measure
echo "Phase 4: Measuring..."
bash measure-after-load.sh
bash analyze-managed-fields.sh

# Phase 5: Test solutions
echo "Phase 5: Testing solutions..."
# Run compression test, etc.

# Phase 6: Collect results
echo "Phase 6: Collecting results..."
bash collect-results.sh

# Cleanup
kill $PROXY_PID

echo "=== Complete ==="
echo "Results in measurements/"
echo "Profiles in profiles/"
```

## Expected Results

Based on our analysis, we expect to observe:
1. **managedFields constitute 25-50% of total object JSON size** for SSA-managed objects
2. **Apiserver RSS scales linearly** with number of SSA-managed objects
3. **Heap profiles show significant allocation** in FieldsV1 deserialization paths
4. **Compression achieves 60-80% reduction** of FieldsV1 raw data
5. **Stripping managedFields would save 20-40%** of apiserver cache memory
