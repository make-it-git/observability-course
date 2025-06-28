### Run

```shell
docker compose up -d --build --force-recreate 
```

### Run simulated load
```shell
k6 run k6/load-test.js
```

### Grafana
```
http://localhost:3000
```

### Stop

```shell
docker compose down -v
```

### Feature toggle

Для создания трейсов, сохраняемых на основе высокого latency, укажите 700/900 (таймаут - 1с)

```shell
curl -X PUT localhost:8081/api/v1/features/add-point-delay-value-start -d '{"value": 700}' -v
curl -X PUT localhost:8081/api/v1/features/add-point-delay-value-end -d   '{"value": 900}' -v
```

Для получения трейсов, сохраняемых на основе ошибок в трейсах, укажите 900/1200 (таймаут - 1с).
Они включат в себя и трейсы на основе высокого latency.

```shell
curl -X PUT localhost:8081/api/v1/features/add-point-delay-value-start -d '{"value": 900}' -v
curl -X PUT localhost:8081/api/v1/features/add-point-delay-value-end -d   '{"value": 1200}' -v
```

Эти значения указывают искуственную latency, создаваемую в сервисе, в миллисекундах.


Добавление метрики threshold
```shell
while true; do curl -d 'threshold{job="track-analyzer-service", status="200"} 100' -X POST 'http://localhost:8428/api/v1/import/prometheus'; sleep 1; done 

```
