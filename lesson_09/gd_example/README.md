## Firing alert for default route

```shell
curl -XPOST http://localhost:9093/api/v2/alerts \
  -H "Content-Type: application/json" \
  -d '[
    {
      "status": "firing",
      "labels": {
        "alertname": "test_alert",
        "severity": "critical",
        "instance": "example.instance"
      },
      "annotations": {
        "summary": "This is a test alert"
      },
      "generatorURL": "http://example.com/prometheus/graph",
      "startsAt": "2025-06-19T22:00:00Z",
      "endsAt": "2025-06-19T22:10:00Z"
    }
  ]'
```

## Firing alert for custom route

```shell
curl -XPOST http://localhost:9093/api/v2/alerts \
  -H "Content-Type: application/json" \
  -d '[
    {
      "status": "firing",
      "labels": {
        "alertname": "my_test_alert",
        "severity": "critical",
        "instance": "example.instance",
        "service": "another-app"
      },
      "annotations": {
        "summary": "This is a test alert"
      },
      "generatorURL": "http://example.com/prometheus/graph",
      "startsAt": "2025-06-19T22:00:00Z",
      "endsAt": "2025-06-19T22:10:00Z"
    }
  ]'
```