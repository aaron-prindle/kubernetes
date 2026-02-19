#!/bin/bash
# generate-ssa-fast.sh
# Fast parallel SSA object generation using background jobs
# Creates N ConfigMaps with M managers, running PARALLEL concurrent kubectl processes

NUM_OBJECTS=${1:-2000}
NUM_MANAGERS=${2:-5}
PARALLEL=${3:-20}
NAMESPACE="ssa-test"

kubectl create namespace $NAMESPACE 2>/dev/null

echo "Creating $NUM_OBJECTS ConfigMaps with $NUM_MANAGERS managers each ($PARALLEL parallel)..."
echo "Total API calls: $((NUM_OBJECTS * NUM_MANAGERS))"
echo "Start time: $(date)"

CREATED=0

apply_object() {
    local i=$1
    local m=$2
    cat <<EOF | kubectl apply --server-side --field-manager="manager-$m" -f - &>/dev/null
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
  key-m${m}-6: "value-${i}-${m}-6"
  key-m${m}-7: "value-${i}-${m}-7"
  key-m${m}-8: "value-${i}-${m}-8"
  key-m${m}-9: "value-${i}-${m}-9"
  key-m${m}-10: "value-${i}-${m}-10"
EOF
}

# Process objects with parallelism
for i in $(seq 1 $NUM_OBJECTS); do
    for m in $(seq 1 $NUM_MANAGERS); do
        apply_object $i $m &

        # Limit parallelism
        while [ $(jobs -r | wc -l) -ge $PARALLEL ]; do
            sleep 0.05
        done
    done

    if [ $((i % 100)) -eq 0 ]; then
        wait  # Wait for all current jobs
        echo "Created $i / $NUM_OBJECTS objects ($(date +%H:%M:%S))"
    fi
done

wait
echo "Done creating $NUM_OBJECTS objects with $NUM_MANAGERS managers each."
echo "End time: $(date)"
