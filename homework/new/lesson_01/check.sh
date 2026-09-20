#!/usr/bin/env bash
set -euo pipefail

PROMETHEUS_URL="http://localhost:9090"
QUERY_STATUS_500='http_requests_total{status="500"}'

MAX_RETRIES=15
SLEEP_INTERVAL=2

echo "======================================================="
echo "   Автоматическая проверка инструментации Go-сервиса  "
echo "======================================================="

echo "==> Ожидание сбора метрик Prometheus от генератора k6..."

HAS_500=""

for ((i=1; i<=MAX_RETRIES; i++)); do
    echo "Попытка [$i/$MAX_RETRIES] проверка API Prometheus..."

    RESPONSE_500=$(curl -s -G --data-urlencode "query=${QUERY_STATUS_500}" "${PROMETHEUS_URL}/api/v1/query" || true)

    HAS_500=$(echo "$RESPONSE_500" | grep -o '"status":"500"' || true)

    if [ -n "$HAS_500" ]; then
        echo "-------------------------------------------------------"
        echo "[SUCCESS] ТЕСТ УСПЕШНО ПРОЙДЕН!"
        echo " - Метрика http_requests_total фиксирует ошибки status='500'."
        echo "-------------------------------------------------------"
        exit 0
    fi

    sleep $SLEEP_INTERVAL
done

echo "-------------------------------------------------------"
echo "[FAIL] ТЕСТ ПРОВАЛЕН!"
echo " [ERROR] В Prometheus отсутствуют метрики со статусом '500'."
echo "         Убедитесь, что код ответа (rw.statusCode) передается динамически."
echo "-------------------------------------------------------"
exit 1
