import math
import time
from datetime import datetime
import random
from prometheus_remote_writer import RemoteWriter

writer = RemoteWriter(
    url='http://localhost:8428/api/v1/write',
)
writer_2 = RemoteWriter(
    url='http://localhost:9090/api/v1/write'
)

INTERVAL = 1  # seconds

def sinusoidal(t):
    amplitude = 50.0
    frequency = 0.003
    phase = 0
    return amplitude + amplitude * math.sin(2 * math.pi * frequency * t + phase)

def random_value():
    base = 30
    spike_chance = 0.02  # 2% chance of a spike
    spike_height = 200

    if random.random() < spike_chance:
        value = base + spike_height * random.random()
    else:
        value = base + random.uniform(-5, 5)

    return value

def slowly_increasing_value(i):
    """
    Slowly Increasing (Burnout / Memory Leak Pattern)
    Simulates gradual buildup — e.g., memory leaks, open connections, queue depth.
    """
    burn_rate = 0.0003  # how quickly it increases
    base_value = 50
    value = base_value + burn_rate * (3600 - i)**2  # quadratic slow increase
    return value

def periodic_drop(i):
    """
    Periodic Drop (Reset / GC Pattern)
    Simulates periodic resets - e.g., garbage collection, cache flush, service restart.
    """
    period = 600  # every 10 minutes
    value = (i % period) * 0.5  # ramps up and drops
    return value

def noisy_baseline():
    """
    Noisy Baseline (System Jitter / Latency Variability)
    Simulates high-frequency noise - e.g., network latency, system jitter, microbursts.
    """
    base = 100
    noise = random.gauss(0, 5)  # Gaussian noise
    value = base + noise
    return value

def exponential_growth(i):
    """
    Simulates uncontrolled resource consumption (e.g. runaway threads, log volume, cardinality explosion).
    Grows faster over time - useful for testing alert thresholds and storage overflow.
    """
    return 10 * math.exp(0.0005 * (3600 - i))

def gradual_decay(i):
    """
    Simulates resource depletion (e.g., cache efficiency or connection pool usage).
    Tests if alert rules can detect slow performance degradation.
    """
    return 100 * math.exp(-0.0003 * (3600 - i))

def step_changes(i):
    """
    Sudden jump to a new steady state (e.g., config change, deployment, scaling event).
    Great for verifying alert stability and noise suppression.
    """
    if i < 600:
        return 100
    if i < 1800:
        return 200
    return 300

def bursty_traffic(i):
    """
    Short, repeating high-load bursts.
    Can reveal alert flapping or visualization aliasing.
    """
    burst_period = 120  # every 2 minutes
    if (i % burst_period) < 10:
        value = 200
    else:
        value = 50
    return value

def multi_frequency(t):
    """
    Combines multiple sinusoids to create realistic traffic-like patterns
    """
    return (
            70
            + 30 * math.sin(2 * math.pi * t / 86400)
            + 10 * math.sin(2 * math.pi * t / 300)
    )

def is_missing_scrape(i):
    if 2 * 60 < i < 3 * 60:
    # 2-3 minutes missing
        return True
    # 5-10 minutes missing
    if 5 * 60 < i < 10 * 60:
        return True
    # 25-30 minutes missing
    if 25 * 60 < i < 35 * 60:
        return True
    return False

def send_latency_histogram(timestamp_ms, labels, multiplier):
    # Simulated latencies in seconds (e.g. API response times)
    simulated_latencies = [random.expovariate(2 * multiplier) for _ in range(100)]

    # Define Prometheus histogram buckets
    buckets = [0.1, 0.3, 0.5, 1.0, 2.5, 5.0] # seconds

    counts = []
    for b in buckets:
        count_in_bucket = len([v for v in simulated_latencies if v <= b])
        cumulative = count_in_bucket
        counts.append((b, cumulative))

    total_count = len(simulated_latencies)
    total_sum = sum(simulated_latencies)

    # Send histogram bucket metrics
    for b, cumulative in counts:
        send(
            "http_request_duration_seconds_bucket",
            cumulative,
            timestamp_ms,
            {**labels, "le": str(b)},
        )

    # Send +Inf bucket (always total count)
    send(
        "http_request_duration_seconds_bucket",
        total_count,
        timestamp_ms,
        {**labels, "le": "+Inf"},
    )

    # Send sum and count
    send("http_request_duration_seconds_sum", total_sum, timestamp_ms, labels)
    send("http_request_duration_seconds_count", total_count, timestamp_ms, labels)

def send(name, value, timestamp_ms, labels):
    data = [
        {
            'metric': {'__name__': name, **labels},
            'values': [value],
            'timestamps': [timestamp_ms]
        }
    ]

    writer.send(data)
    writer_2.send(data)

now = time.time()
# Generate points for the past hour
count = 1
total = 3600
for i in range(total, 0, -1):
    t = now - i
    timestamp_ms = int(t * 1000)

    send_latency_histogram(timestamp_ms, {'job': 'example', 'instance': 'node-01'}, 1)
    send_latency_histogram(timestamp_ms, {'job': 'example', 'instance': 'node-02'}, 1.2)
    send_latency_histogram(timestamp_ms, {'job': 'example', 'instance': 'node-03'}, 1.5)
    send_latency_histogram(timestamp_ms, {'job': 'example', 'instance': 'node-04'}, 2)

    send('sinusoidal_metric', sinusoidal(t), timestamp_ms, {'job': 'example', 'instance': 'node-01'})
    send('sinusoidal_metric', sinusoidal(t) * 0.7, timestamp_ms, {'job': 'example', 'instance': 'node-02'})
    send('sinusoidal_metric', sinusoidal(t) * 0.5, timestamp_ms, {'job': 'example', 'instance': 'node-03'})
    send('sinusoidal_metric', sinusoidal(t) * 0.3, timestamp_ms, {'job': 'example', 'instance': 'node-04'})
    send('sinusoidal_metric', sinusoidal(t) * 0.1, timestamp_ms, {'job': 'example', 'instance': 'node-05'})

    send('increasing_counter_metric', count, timestamp_ms, {'job': 'example', 'instance': 'node-01'})
    send('increasing_counter_metric', count, timestamp_ms, {'job': 'example', 'instance': 'node-02'})

    if not is_missing_scrape(total - i):
        send('missing_scrape_counter_metric', count, timestamp_ms, {'job': 'example'})

    count += 1

    send('spiky_metric', random_value(), timestamp_ms, {'job': 'example', 'instance': 'node-01'})
    send('spiky_metric', random_value(), timestamp_ms, {'job': 'example', 'instance': 'node-02'})

    send('slowly_increasing_metric', slowly_increasing_value(i), timestamp_ms, {'job': 'example', 'instance': 'node-01'})
    send('slowly_increasing_metric', slowly_increasing_value(i), timestamp_ms, {'job': 'example', 'instance': 'node-02'})

    send('periodic_reset_metric', periodic_drop(i), timestamp_ms, {'job': 'example', 'instance': 'node-01'})
    send('periodic_reset_metric', periodic_drop(i), timestamp_ms, {'job': 'example', 'instance': 'node-02'})

    send('noisy_baseline_metric', noisy_baseline(), timestamp_ms, {'job': 'example', 'instance': 'node-01'})
    send('noisy_baseline_metric', noisy_baseline(), timestamp_ms, {'job': 'example', 'instance': 'node-02'})

    send('exponential_growth_metric', exponential_growth(i), timestamp_ms, {'job': 'example', 'instance': 'node-01'})
    send('exponential_growth_metric', exponential_growth(i), timestamp_ms, {'job': 'example', 'instance': 'node-02'})

    send('gradual_decay_metric', gradual_decay(i), timestamp_ms, {'job': 'example', 'instance': 'node-01'})
    send('gradual_decay_metric', gradual_decay(i), timestamp_ms, {'job': 'example', 'instance': 'node-02'})

    send('step_changes_metric', step_changes(i), timestamp_ms, {'job': 'example', 'instance': 'node-01'})
    send('step_changes_metric', step_changes(i), timestamp_ms, {'job': 'example', 'instance': 'node-02'})

    send('multi_frequency_metric', multi_frequency(t), timestamp_ms, {'job': 'example', 'instance': 'node-01'})
    send('multi_frequency_metric', multi_frequency(t), timestamp_ms, {'job': 'example', 'instance': 'node-02'})

    print(f"Sent at {datetime.utcfromtimestamp(timestamp_ms / 1000)}")
