#!/bin/bash
# generate-ssa-objects.sh
# Creates N ConfigMaps using SSA with M different managers
# Each manager applies different keys, generating substantial managedFields

NUM_OBJECTS=${1:-1000}
NUM_MANAGERS=${2:-5}
NAMESPACE="ssa-test"

kubectl create namespace $NAMESPACE 2>/dev/null

echo "Creating $NUM_OBJECTS ConfigMaps with $NUM_MANAGERS managers each..."
echo "Start time: $(date)"

for i in $(seq 1 $NUM_OBJECTS); do
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

    if [ $((i % 50)) -eq 0 ]; then
        echo "Created $i / $NUM_OBJECTS objects ($(date +%H:%M:%S))"
    fi
done

echo "Done creating $NUM_OBJECTS objects with $NUM_MANAGERS managers each."
echo "End time: $(date)"
