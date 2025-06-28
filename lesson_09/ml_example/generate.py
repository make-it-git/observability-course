import math
import time
from prometheus_remote_writer import RemoteWriter

# Configure RemoteWriter for your Prometheus instance
writer = RemoteWriter(
    url='http://localhost:9090/api/v1/write',
)

# Sinusoidal parameters
AMPLITUDE = 1.0
FREQUENCY = 0.03
PHASE = 0
INTERVAL = 1  # seconds

while True:
    t = time.time()
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
    print(f"Sent value {value:.4f} at {timestamp_ms}")
    time.sleep(INTERVAL)