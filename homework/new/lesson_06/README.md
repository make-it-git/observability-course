# Домашнее задание: Трейсинг "по-взрослому"

### Цель

Научиться настраивать `otel-collector` в связке c другими коллекторами для эффективного сбора и фильтрации трейсов, минимизируя потери данных и обеспечивая видимость проблемных запросов.

---

### Задача

Проект находится в репозитории: [https://github.com/make-it-git/observability-course/tree/main/homework/new/lesson_06/project](https://github.com/make-it-git/observability-course/tree/main/homework/new/lesson_06/project)

На занятии вы познакомились с проектом, состоящим из 2-х сервисов:

* `driver-location-service`
* `track-analyzer-service`

В файле проекта `infra/otel/otel-collector-config.yaml` уже есть рабочая конфигурация для `opentelemetry-collector`. Однако трейсы сохраняются с определенной вероятностью в силу того, что в продовых окружениях сэмплирование — необходимость, чтобы не тратить половину бюджета проекта на хранение трейсов 🙂

**Вам нужно:**

1. Убрать сэмплирование «основного» `otel-collector` (который уже настроен).
2. Настроить tail sampling для «агрегирующего» `otel-collector` на основе latency трейса и ошибок.

---

### Архитектура потока данных

```text
application
 └──> otel-collector sidecar (присылает 100% трейсов)
       └──> otel-collector aggregator (получает 100% трейсов и сэмплирует по latency/ошибкам)
             └──> Storage (Jaeger / Tempo / etc.)

```

---

### Зачем так нужно делать?

Трейсинг — очень затратная операция с точки зрения хранения и обработки. Нагрузка в 1000 RPS на ваш сервис может легко генерировать $N$ килобайт данных на один трейс. Умножаем на 1000 и получаем $1000 \times N$ килобайт каждую секунду.

#### Как это решается?

Нам не нужно в общем случае хранить вообще все трейсы. Хранение 100% данных — не общепринятая практика из-за высокого потребления ресурсов при больших нагрузках.

> **Пример:**
> В [статье OzonTech на Хабре](https://habr.com/en/companies/ozontech/articles/708274/) упоминается подход с сохранением 100% трейсов только за ограниченный промежуток времени (10–15 минут) с последующим умным сэмплированием по настроенным параметрам и вычислением критического пути.

#### Скейлинг и структура коллекторов

Один коллектор не выдержит всей нагрузки. В Kubernetes типовая схема выглядит так:

* **Слой 1 (Sidecar / DaemonSet):**
* *Option A:* Sidecar `otel-collector` (один коллектор на каждый инстанс сервиса в поде).
* *Option B:* `otel-collector` как `DaemonSet` (один коллектор на ноду кластера — более экономный путь по ресурсам).


* **Слой 2 (Aggregator):**
  Отдельный агрегирующий коллектор, принимающий телеметрию со всех коллекторов первого слоя. Его задача — применить правила сэмплирования и решить, отправлять ли трейс в итоговое хранилище.

**Базовые критерии сохранения трейса в агрегаторе:**

* В трейсе есть ошибки (`status.code == ERROR`).
* Трейс показывает высокую latency распределенного процесса.
* Избежание «рваных» трейсов: если сэмплировать на уровне sidecar, фрагменты одного и того же трейса с разных сервисов могут быть частично отброшены.

---

### Файлы для решения

Готовые заготовки конфигурации находятся в проекте:

* `docker-compose.yml`
* `infra/otel/otel-collector-config-sidecar.yaml`
* `infra/otel/otel-collector-config-aggregator.yaml`

Для создания нагрузки используйте:

```bash
k6 run k6/load-test.js

```

---

### Полезные ссылки

* [OTTL Span Context Reference](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl/contexts/ottlspan) — атрибуты для tail sampling processor.
* [OpenTelemetry Blog: Tail Sampling](https://opentelemetry.io/blog/2022/tail-sampling/) — концепция и лучшие практики.
* [Tail Sampling Processor Configuration](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/tailsamplingprocessor) — документация процессора.

#### ⭐ Задание со звездочкой (опционально)

* [Load Balancing Exporter](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/loadbalancingexporter) — балансировка и масштабирование коллекторов.
* [OpenTelemetry Collector Scaling](https://opentelemetry.io/docs/collector/scaling/) — руководства по масштабированию.