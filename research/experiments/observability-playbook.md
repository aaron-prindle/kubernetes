# Observability Playbook for SSA Memory Profiling

## Metrics to Collect
- `process_resident_memory_bytes` (kube-apiserver)
- `go_memstats_heap_inuse_bytes`
- request metrics split by verb/resource (especially `APPLY`)
- watch/list latency histograms

## pprof Profiles
Capture at least:
- `heap`
- `allocs`
- `profile` (CPU)

Example:
```bash
# endpoint may vary by setup
curl -sS "http://127.0.0.1:6060/debug/pprof/heap" > heap.pb.gz
curl -sS "http://127.0.0.1:6060/debug/pprof/allocs" > allocs.pb.gz
curl -sS "http://127.0.0.1:6060/debug/pprof/profile?seconds=60" > cpu.pb.gz
```

Inspect:
```bash
go tool pprof -top heap.pb.gz
go tool pprof -top allocs.pb.gz
go tool pprof -top cpu.pb.gz
```

## API payload sizing samples
```bash
kubectl get configmap -n ssa-lab cm-1 -o json > cm.json
jq '.metadata.managedFields' cm.json > managedfields.json
wc -c cm.json managedfields.json
```

## Useful comparative snapshots
Take snapshots at:
1. empty cluster baseline
2. after object creation
3. after SSA churn
4. after delete + settle period

## Interpretation Guide
- If RSS remains elevated after delete, inspect retained heap paths and cache structures.
- If apply CPU/allocs dominate, focus on decode/merge/encode and conversion hotspots.
- If list/watch latency spikes with high object counts, analyze response path amplification from metadata size.
