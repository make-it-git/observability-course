
## Cardinality explorer

```
http://localhost:8428/vmui/?#/cardinality
```

## Trigger deployment from pipeline

```shell
export PORT=2003
export SERVER=localhost
echo "deployments.production.service.track-analyzer-service 1 `date +%s`" | nc ${SERVER} ${PORT}
export current_time=$(date +%s)
export seconds_ago=$((60 * 40))
export past_time=$((current_time - seconds_ago))
echo "deployments.production.service.driver-location-service 1 $past_time" | nc ${SERVER} ${PORT}
```

## Create fake load

```shell
ab -n 1000000 -c 10 localhost:8080/metrics
```

## View exemplars

```shell
curl -H 'Accept: application/openmetrics-text' localhost:8080/metrics | grep http_request_duration_seconds_bucket
```

## Feature toggles

### Add fake delay to "add point" feature

```shell
curl -X PUT localhost:8081/api/v1/features/add-point-delay-value-start -d '{"value": 700}' -v
curl -X PUT localhost:8081/api/v1/features/add-point-delay-value-end -d   '{"value": 900}' -v
```


### Add fake delay to "track analysis" feature
```shell
curl -X PUT localhost:8081/api/v1/features/slow-save-track-analysis -d '{"value": true}' -v
```


