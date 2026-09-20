#!/usr/bin/env bash
set -euo pipefail

PROMETHEUS_URL="http://localhost:9090"
MAX_RETRIES=10
RETRY_INTERVAL=3

echo "=== Запуск проверки выполнения домашнего задания ==="

# 1. Проверка доступности Prometheus API
echo "[1/3] Проверка доступности Prometheus..."
for ((i=1; i<=MAX_RETRIES; i++)); do
  if curl -s "${PROMETHEUS_URL}/-/healthy" | grep -q "Prometheus Server is Healthy"; then
    echo "Prometheus доступен."
    break
  fi
  if [ "$i" -eq "$MAX_RETRIES" ]; then
    echo "ОШИБКА: Prometheus недоступен по адресу ${PROMETHEUS_URL}"
    exit 1
  fi
  echo "Ожидание Prometheus ($i/$MAX_RETRIES)..."
  sleep $RETRY_INTERVAL
done

# 2. Проверка наличия метрик ошибок status="503" в гистограмме
echo "[2/3] Проверка сбора метрик http_request_duration_seconds_bucket со статусом 503..."
QUERY_503='http_request_duration_seconds_bucket{status="503"}'
SUCCESS=0

for ((i=1; i<=MAX_RETRIES; i++)); do
  RESPONSE=$(curl -s -G --data-urlencode "query=${QUERY_503}" "${PROMETHEUS_URL}/api/v1/query")
  RESULT_COUNT=$(echo "$RESPONSE" | grep -o '"result":\[[^]]*\]' | grep -o 'metric' | wc -l || true)

  if [ "$RESULT_COUNT" -gt 0 ]; then
    echo "Метрики гистограммы со статусом 503 успешно найдены!"
    SUCCESS=1
    break
  fi

  echo "Метрики со статусом 503 пока не найдены в гистограмме. Ожидание накопления ($i/$MAX_RETRIES)..."
  sleep $RETRY_INTERVAL
done

if [ "$SUCCESS" -ne 1 ]; then
  echo "ОШИБКА: В метрике http_request_duration_seconds_bucket отсутствуют записи со статусом 503."
  echo "Проверьте, исправлен ли файл main.go (передается ли rw.statusCode в Observe)."
  exit 1
fi

# 3. Валидация вычисления histogram_quantile p95
echo "[3/3] Проверка работы PromQL-запроса histogram_quantile (p95 latency)..."
PROMQL_P95='histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[1m])) by (le, path))'

RESPONSE_P95=$(curl -s -G --data-urlencode "query=${PROMQL_P95}" "${PROMETHEUS_URL}/api/v1/query")
VALUE=$(echo "$RESPONSE_P95" | grep -o '"value":\[[^]]*\]' | head -n1 || true)

if [ -n "$VALUE" ]; then
  echo "PromQL-запрос вычисления p95 успешно выполнен. Результат: ${VALUE}"
  echo "=== ПРОВЕРКА УСПЕШНО ПРОЙДЕНА (Exit 0) ==="
  exit 0
else
  echo "ОШИБКА: Не удалось получить корректное значение p95 через histogram_quantile."
  echo "Ответ Prometheus: ${RESPONSE_P95}"
  exit 1
fi
