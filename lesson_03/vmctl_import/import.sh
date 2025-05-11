#!/bin/bash
if [ -z "$TERM" ]; then
  export TERM=xterm-256color
fi

set -e
apk update && apk add --no-cache curl victoria-metrics-tools && echo 'Importing data...'

#  Create the snapshot directory.
prometheus_data_dir="/prometheus"
snapshot_name=$(date +%s)
snapshot_dir="$prometheus_data_dir/snapshots/$snapshot_name"

# Check if Prometheus is healthy; fail if it's not
curl -fs http://prometheus:9090/graph || (echo "Prometheus not healthy" && exit 1)

echo "****************"
echo "Creating Prometheus snapshot..."
# Create a Prometheus snapshot via API
curl -X POST http://prometheus:9090/api/v1/admin/tsdb/snapshot
echo "****************"

# Find the latest created snapshot
latest_snapshot=$(ls -td "$prometheus_data_dir/snapshots/"*/ | head -n 1)

if [ -z "$latest_snapshot" ]; then
  echo "Error: No snapshots found"
  exit 1
fi

latest_snapshot_dir=$(basename "$latest_snapshot")

echo "****************"
echo "Latest snapshot directory: $latest_snapshot_dir"
echo "****************"

#  Import the snapshot.
echo "****************"
echo "Importing Prometheus snapshot with vmctl..."
vmctl prometheus --prom-snapshot=/prometheus/snapshots/"$latest_snapshot_dir"/ --vm-addr=http://victoria-metrics:8428 --verbose -s --disable-progress-bar

echo "Done!"
echo "****************"
