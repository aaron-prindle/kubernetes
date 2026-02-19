#!/bin/bash
# analyze-managed-fields.sh
# Analyzes the size contribution of managedFields across all ConfigMaps

NAMESPACE="${1:-ssa-test}"

echo "=== ManagedFields Size Analysis ==="
echo "Namespace: $NAMESPACE"

kubectl get configmaps -n $NAMESPACE -o json | \
    python3 -c "
import json, sys
data = json.load(sys.stdin)
total_mf_size = 0
total_obj_size = 0
max_mf_size = 0
max_mf_obj = ''
entry_counts = []

for item in data['items']:
    obj_json = json.dumps(item)
    total_obj_size += len(obj_json)
    mf = item.get('metadata', {}).get('managedFields', [])
    mf_json = json.dumps(mf)
    mf_size = len(mf_json)
    total_mf_size += mf_size
    entry_counts.append(len(mf))
    if mf_size > max_mf_size:
        max_mf_size = mf_size
        max_mf_obj = item['metadata']['name']

num_objects = len(data['items'])
print(f'Total objects: {num_objects}')
print(f'Total object size (JSON): {total_obj_size / 1024 / 1024:.2f} MB')
print(f'Total managedFields size: {total_mf_size / 1024 / 1024:.2f} MB')
print(f'managedFields as % of total: {total_mf_size / total_obj_size * 100:.1f}%')
print(f'Average managedFields per object: {total_mf_size / num_objects / 1024:.2f} KB')
print(f'Average entries per object: {sum(entry_counts) / len(entry_counts):.1f}')
print(f'Largest managedFields: {max_mf_size / 1024:.2f} KB ({max_mf_obj})')

# Without managedFields
stripped_size = total_obj_size - total_mf_size
print(f'')
print(f'=== Size Comparison ===')
print(f'With managedFields:    {total_obj_size / 1024 / 1024:.2f} MB')
print(f'Without managedFields: {stripped_size / 1024 / 1024:.2f} MB')
print(f'Savings from stripping: {total_mf_size / 1024 / 1024:.2f} MB ({total_mf_size / total_obj_size * 100:.1f}%)')
"
