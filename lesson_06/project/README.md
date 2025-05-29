http://localhost:8428/vmui/?#/cardinality

###

export PORT=2003
export SERVER=localhost
echo "deployments.production.service.track-analyzer-service 1 `date +%s`" | nc ${SERVER} ${PORT}
export current_time=$(date +%s)
export seconds_ago=$((60 * 40))
export past_time=$((current_time - seconds_ago))
echo "deployments.production.service.driver-location-service 1 $past_time" | nc ${SERVER} ${PORT}

###

ab -n 1000000 -c 10 localhost:8080/metrics

###

curl -H 'Accept: application/openmetrics-text' localhost:8080/metrics | grep http_request_duration_seconds_bucket

###

curl -X PUT localhost:8081/api/v1/features/add-point-delay-value-start -d '{"value": 700}' -v
curl -X PUT localhost:8081/api/v1/features/add-point-delay-value-end -d   '{"value": 900}' -v


###

curl -X PUT localhost:8081/api/v1/features/slow-save-track-analysis -d '{"value": true}' -v


