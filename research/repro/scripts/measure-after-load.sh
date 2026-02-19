#!/bin/bash
# measure-after-load.sh
# Takes post-load memory measurements

PROFILES_DIR="${1:-/Users/aaronprindle/kubernetes/research/repro/profiles}"
MEASUREMENTS_DIR="${2:-/Users/aaronprindle/kubernetes/research/repro/measurements}"

echo "=== Post-Load Memory Measurement ==="

# Docker stats
docker stats --no-stream $(docker ps --filter "name=ssa-memory-test-control-plane" -q)

# Heap profile
curl -s http://localhost:8001/debug/pprof/heap > "$PROFILES_DIR/postload-heap.prof"
echo ""
echo "Heap profile saved to $PROFILES_DIR/postload-heap.prof"

# Top allocators
echo ""
echo "=== Top Memory Allocators ==="
go tool pprof -text "$PROFILES_DIR/postload-heap.prof" 2>&1 | head -40
go tool pprof -text "$PROFILES_DIR/postload-heap.prof" 2>&1 | head -40 > "$MEASUREMENTS_DIR/postload-top-allocators.txt"

# Search for managedFields-related allocations
echo ""
echo "=== ManagedFields-Related Allocations ==="
go tool pprof -text "$PROFILES_DIR/postload-heap.prof" 2>&1 | grep -i -E "managed|field|FieldsV1|ObjectMeta|merge|Set"
