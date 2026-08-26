import math
import time
from datetime import datetime
from prometheus_remote_writer import RemoteWriter

# Configure RemoteWriter for your Prometheus instance
writer = RemoteWriter(
    url='http://localhost:8428/api/v1/write',
)

# Sinusoidal parameters
AMPLITUDE = 1.0
FREQUENCY = 0.003
PHASE = 0
INTERVAL = 1  # seconds

now = time.time()
# Generate points for the past hour
# Remote write like this does not work for prometheus, but works for victoria
# https://github.com/VictoriaMetrics/VictoriaMetrics/issues/827
# Error while inserting data more then 2 days in future.
count = 1
for i in range(3600, 0, -1):
    t = now - i
    value = AMPLITUDE * math.sin(2 * math.pi * FREQUENCY * t + PHASE)
    timestamp_ms = int(t * 1000)

    # Prepare data in the format expected by prometheus-remote-writer
    data = [
        {
            'metric': {'__name__': 'http_request_latency', 'job': 'example'},
            'values': [value],
            'timestamps': [timestamp_ms]
        }
    ]

    writer.send(data)
    print(f"Sent value {value:.4f} at {datetime.utcfromtimestamp(timestamp_ms / 1000)}")

    data = [
        {
            'metric': {'__name__': 'http_request_total', 'job': 'example'},
            'values': [count],
            'timestamps': [timestamp_ms]
        }
    ]
    count += 1

    writer.send(data)