# Домашнее задание: Коррекция инструментации HTTP-сервиса и поиск скрытых 5xx ошибок

> В самом конце этого файла находится раздел **3. РЕШЕНИЕ**. Не листайте вниз до тех пор, пока не попробуете решить задачу самостоятельно!

---

## 1. ЗАДАНИЕ

### Цель

Научиться правильно инструментаровать HTTP-сервис на Go с использованием официального SDK Prometheus (`client_golang`). Вы освоите работу со счетчиками (`prometheus.CounterVec`), научитесь динамически подставлять HTTP status codes в лейблы метрик и находить ошибки инструментации, из-за которых мониторинг показывает идеальное состояние системы во время реального сбоя.

---

### Контекст / Легенда

Вы приступаете к дежурству в качестве инженера в команде сервиса обработки заказов `order-processor`. Сервис принимает запросы клиентов на создание заказов и перед подтверждением обращается к внешнему платежному шлюзу.

От отдела поддержки клиентов поступил срочный тикет: пользователи массово жалуются на ошибки `500 Internal Server Error` при попытке оформить заказ.

Однако дежурный инженер утверждал, что на дашбордах всё в порядке: метрика `http_requests_total` показывает **100% успешных ответов (`status="200"`)**, а метрики с 5xx ошибками отсутствуют.

При проведении аудита кода выяснилось, что предыдущий разработчик в файле `main.go` реализовывал HTTP Middleware и захардкодил передачу строкового значения `"200"` в лейбл `status` метрики `http_requests_total`. В итоге, даже когда внешняя система падает и сервис возвращает HTTP 500, счетчик увеличивает количество якобы успешных ответов со статусом 200.

---

### Инструкция

#### 1. Запуск тестового стенда

Разверните сервис, Prometheus, Grafana и генератор нагрузки k6 одной командой:

```bash
docker compose up -d

```

#### 2. Просмотр метрик в интерфейсах

После запуска контейнеров генератор нагрузки `k6` начинает отправлять запросы. Вы можете изучить текущее состояние метрик:

* **Prometheus UI:** [click](http://localhost:9090/query?g0.expr=http_requests_total&g0.show_tree=0&g0.tab=table&g0.range_input=1h&g0.res_type=auto&g0.res_density=medium&g0.display_mode=lines&g0.show_exemplars=0)
* **Grafana Explore:** [click](http://localhost:3000/explore?left=%5B%22now-1h%22,%22now%22,%22Prometheus%22,%7B%22expr%22:%22http_requests_total%22%7D%5D). (Логин/пароль в Grafana не требуются, включен анонимный доступ с правами Admin).

#### 3. Диагностика и исправление

Изучите файл `main.go` и найдите место в `metricsMiddleware`, где фиксируются метрики. Исправьте код так, чтобы в лейбл `status` передавался **реальный код ответа** (`rw.statusCode`), переведенный в строку.

#### 4. Пересборка и применение

После внесения изменений в `main.go` пересоберите и перезапустите контейнер приложения:

```bash
docker compose up -d --build app

```

#### 5. Автоматическая проверка

Запустите скрипт проверки решения:

```bash
bash check.sh

```

---

### Подсказки / Notes

* В Go для конвертации числового HTTP-статуса (`int`) в строку используйте функцию `strconv.Itoa(code)`.
* Вспомогательная структура `responseWriter` в `main.go` перехватывает вызов `WriteHeader(code)` и сохраняет его в поле `statusCode`.
* Проверить появление метрик ошибок в Prometheus можно запросом PromQL:
```promql
http_requests_total{status="500"}
```
---

---

---

---

---

---

---

---

---

---

---

---

---

---

---

---

---

---

---

---

---

---

---

---

---

---

---

---

---

---

## 3. РЕШЕНИЕ

### В чем заключалась ошибка

В функции `metricsMiddleware` вызов инкремента счетчика `httpRequestsTotal.WithLabelValues(...)` содержал статическое значение `"200"`:

```go
// НЕВЕРНО:
httpRequestsTotal.WithLabelValues(r.Method, handlerName, "200").Inc()

```

Даже когда обработчик `processOrderHandler` возвращал ошибку `w.WriteHeader(http.StatusInternalServerError)` (код 500), счетчик увеличивал значение только для комбинации лейблов `status="200"`.

---

### Исправление `main.go`

```go
func metricsMiddleware(next http.HandlerFunc, handlerName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next(rw, r)

		// ИСПРАВЛЕНИЕ: Преобразуем перехваченный rw.statusCode в строку с помощью strconv.Itoa
		statusStr := strconv.Itoa(rw.statusCode)
		httpRequestsTotal.WithLabelValues(r.Method, handlerName, statusStr).Inc()
	}
}
```
