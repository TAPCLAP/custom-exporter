# custom-exporter

Prometheus-экспортёр с набором независимых коллекторов. Каждый коллектор включается и выключается отдельно в YAML-конфиге. Метрики отдаются по HTTP на эндпоинте `/metrics`.

## Запуск

```bash
custom-exporter --config /path/to/config.yaml
```

Или через переменные окружения:

| Переменная | Описание | По умолчанию |
|---|---|---|
| `CONFIG` | Путь к файлу конфигурации | — (обязательно) |
| `HOST` | Адрес для прослушивания | `127.0.0.1` |
| `PORT` | Порт для прослушивания | `8200` |

Флаги:

- `--config` — путь к конфигу (альтернатива `CONFIG`)
- `--version` — вывести версию и завершить работу

## HTTP-эндпоинты

| Путь | Описание |
|---|---|
| `/metrics` | Метрики Prometheus |
| `/live`, `/livez` | Liveness probe (HTTP 200) |
| `/ready`, `/readyz` | Readiness probe (HTTP 200) |

## Конфигурация

Конфиг — YAML-файл. Все коллекторы независимы: регистрируются и запускаются только при `enabled: true`.

```yaml
fileHashCollector:
  enabled: false
  files: []

portCollector:
  enabled: false
  targets: []

processCollector:
  enabled: false
  processes: []

systemCollector:
  enabled: false

puppetCollector:
  enabled: false
  lastRunReportPath: /opt/puppetlabs/puppet/cache/state/last_run_report.yaml

hetznerCollector:
  enabled: false
  randomSleepBeforeStart: false

hetznerCloudCollector:
  enabled: false
  randomSleepBeforeStart: false

yandexCloudCollector:
  enabled: false
  randomSleepBeforeStart: false

awsCloudCollector:
  enabled: false
  randomSleepBeforeStart: false

tlsCertificateCollector:
  enabled: false
  excludeCA: false
  directories: []
  files: []
```

---

## Коллекторы

### fileHashCollector

Вычисляет CRC32 (IEEE) содержимого указанных файлов. Позволяет отслеживать изменения файлов на диске.

**Интервал обновления:** 180 секунд.

**Конфиг:**

| Параметр | Тип | Описание |
|---|---|---|
| `enabled` | bool | Включить коллектор |
| `files` | []string | Список путей к файлам |

**Пример:**

```yaml
fileHashCollector:
  enabled: true
  files:
    - /etc/nginx/nginx.conf
    - /etc/hosts
```

**Метрики:**

| Метрика | Тип | Лейблы | Описание |
|---|---|---|---|
| `file_hash` | Gauge | `file` | CRC32-хеш содержимого файла. Если файл отсутствует — `0` |

---

### portCollector

Проверяет доступность TCP/UDP-портов на удалённых хостах.

**Интервал обновления:** 60 секунд. Таймаут подключения — 5 секунд.

**Конфиг:**

| Параметр | Тип | Описание |
|---|---|---|
| `enabled` | bool | Включить коллектор |
| `targets` | []object | Список целей для проверки |

Поля цели (`targets`):

| Поле | Тип | Описание |
|---|---|---|
| `host` | string | Хост или IP |
| `port` | uint16 | Порт |
| `protocol` | string | `TCP` или `UDP` |

**Пример:**

```yaml
portCollector:
  enabled: true
  targets:
    - host: example.com
      port: 443
      protocol: TCP
    - host: 10.0.0.1
      port: 53
      protocol: UDP
```

**Метрики:**

| Метрика | Тип | Лейблы | Описание |
|---|---|---|---|
| `network_target` | Gauge | `host`, `port`, `protocol` | `1` — порт доступен, `0` — недоступен |

---

### processCollector

Собирает метрики по процессам Linux через `/proc`. Считает общее количество процессов, распределение по состояниям, а также агрегирует CPU time и resident memory для процессов, отфильтрованных по regex.

**Интервал обновления:** 15 секунд.

**Конфиг:**

| Параметр | Тип | Описание |
|---|---|---|
| `enabled` | bool | Включить коллектор |
| `processes` | []object | Список фильтров процессов |

Поля фильтра (`processes`):

| Поле | Тип | Описание |
|---|---|---|
| `process` | string | Имя метрики (лейбл `process`) |
| `regex` | string | Regex для сопоставления с полем `comm` процесса в `/proc` |

**Пример:**

```yaml
processCollector:
  enabled: true
  processes:
    - process: nginx
      regex: "^nginx$"
    - process: postgres
      regex: "^postgres$"
```

**Метрики:**

| Метрика | Тип | Лейблы | Описание |
|---|---|---|---|
| `processes_count` | Gauge | `type` | Количество процессов. `type=all` — всего; остальные значения: `running`, `sleeping`, `zombie`, `stopped` и др. |
| `process_running_status` | Gauge | `process` | `1` — хотя бы один процесс с данным именем запущен, `0` — нет |
| `process_memory_resident` | Gauge | `process` | Суммарная resident memory (байты) для процессов, совпавших с regex |
| `process_cpu_time_total` | Counter | `process` | Суммарное CPU time (jiffies) для процессов, совпавших с regex |

---

### systemCollector

Системные метрики хоста: hostname, uptime, количество залогиненных пользователей, контрольные суммы hostname и `uname`.

**Интервал обновления:** 60 секунд.

**Конфиг:**

| Параметр | Тип | Описание |
|---|---|---|
| `enabled` | bool | Включить коллектор |

**Метрики:**

| Метрика | Тип | Лейблы | Описание |
|---|---|---|---|
| `hostname` | Gauge | `hostname` | Всегда `1`; лейбл содержит текущий hostname |
| `hostname_checksum` | Gauge | — | CRC32 hostname |
| `uname_checksum` | Gauge | — | CRC32 вывода `uname -a` |
| `uptime_seconds` | Counter | — | Uptime системы в секундах (из `/proc/uptime`) |
| `login_users_count` | Gauge | — | Количество залогиненных пользователей (`who -q`) |

---

### puppetCollector

Читает отчёт последнего запуска Puppet Agent и экспортирует статус компиляции каталога.

**Интервал обновления:** 300 секунд.

**Конфиг:**

| Параметр | Тип | Описание |
|---|---|---|
| `enabled` | bool | Включить коллектор |
| `lastRunReportPath` | string | Путь к `last_run_report.yaml`. По умолчанию: `/opt/puppetlabs/puppet/cache/state/last_run_report.yaml` |

**Пример:**

```yaml
puppetCollector:
  enabled: true
  lastRunReportPath: /opt/puppetlabs/puppet/cache/state/last_run_report.yaml
```

**Метрики:**

| Метрика | Тип | Лейблы | Описание |
|---|---|---|---|
| `puppet_catalog_last_compile_timestamp` | Gauge | — | Значение `configuration_version` из отчёта |
| `puppet_catalog_last_compile_status` | Gauge | — | `1` — `transaction_completed: true`, `0` — иначе |

---

### hetznerCollector

Получает список выделенных серверов через Hetzner Robot API.

**Интервал обновления:** 600 секунд.

**Переменные окружения (обязательны при включении):**

| Переменная | Описание |
|---|---|
| `HROBOT_USER` | Логин Robot API |
| `HROBOT_PASS` | Пароль Robot API |

**Конфиг:**

| Параметр | Тип | Описание |
|---|---|---|
| `enabled` | bool | Включить коллектор |
| `randomSleepBeforeStart` | bool | Случайная задержка 1–60 секунд перед первым запуском |

**Метрики:**

| Метрика | Тип | Лейблы | Описание |
|---|---|---|---|
| `hetzner_robot_server` | Gauge | `id`, `name`, `type`, `zone`, `region`, `ip` | Наличие сервера (`1`) |

---

### hetznerCloudCollector

Получает список серверов Hetzner Cloud.

**Интервал обновления:** 600 секунд.

**Переменные окружения (один из вариантов):**

- `HCLOUD_TOKEN` — один аккаунт
- `HCLOUD_TOKEN_0`, `HCLOUD_TOKEN_1`, … — несколько аккаунтов

**Конфиг:**

| Параметр | Тип | Описание |
|---|---|---|
| `enabled` | bool | Включить коллектор |
| `randomSleepBeforeStart` | bool | Случайная задержка 1–60 секунд перед первым запуском |

**Метрики:**

| Метрика | Тип | Лейблы | Описание |
|---|---|---|---|
| `hetzner_cloud_server` | Gauge | `id`, `name`, `type`, `zone`, `region`, `ip` | Наличие сервера (`1`) |

---

### yandexCloudCollector

Получает список виртуальных машин Yandex Cloud.

**Интервал обновления:** 600 секунд. Метрики удалённых ВМ очищаются через 30 минут после последнего обновления.

**Переменные окружения (один аккаунт):**

| Переменная | Описание |
|---|---|
| `YANDEX_CLOUD_SERVICE_ACCOUNT_ID` | ID сервисного аккаунта |
| `YANDEX_CLOUD_SERVICE_ACCOUNT_KEY_ID` | ID ключа |
| `YANDEX_CLOUD_SERVICE_ACCOUNT_PRIVATE_KEY` | Приватный ключ (base64) |
| `YANDEX_CLOUD_SERVICE_ACCOUNT_FOLDER_ID` | ID каталога |

Для нескольких аккаунтов — те же переменные с суффиксами `_0`, `_1`, …

**Конфиг:**

| Параметр | Тип | Описание |
|---|---|---|
| `enabled` | bool | Включить коллектор |
| `randomSleepBeforeStart` | bool | Случайная задержка 1–60 секунд перед первым запуском |

**Метрики:**

| Метрика | Тип | Лейблы | Описание |
|---|---|---|---|
| `yandex_cloud_server` | Gauge | `id`, `name`, `type`, `zone`, `region`, `public_ip`, `private_ip`, `cpu_count`, `memory` | Наличие ВМ (`1`) |

---

### awsCloudCollector

Получает список EC2-инстансов AWS.

**Интервал обновления:** 600 секунд.

**Переменные окружения (один аккаунт):**

| Переменная | Описание |
|---|---|
| `AWS_ACCESS_KEY_ID` | Access Key ID |
| `AWS_SECRET_ACCESS_KEY` | Secret Access Key |
| `AWS_DEFAULT_REGION` | Регион по умолчанию |

Для нескольких аккаунтов — `AWS_ACCESS_KEY_ID_0`, `AWS_SECRET_ACCESS_KEY_0`, `AWS_DEFAULT_REGION_0`, …

**Конфиг:**

| Параметр | Тип | Описание |
|---|---|---|
| `enabled` | bool | Включить коллектор |
| `randomSleepBeforeStart` | bool | Случайная задержка 1–60 секунд перед первым запуском |

**Метрики:**

| Метрика | Тип | Лейблы | Описание |
|---|---|---|---|
| `aws_cloud_server` | Gauge | `id`, `name`, `type`, `zone`, `region`, `public_ip`, `private_ip`, `private_dns_name` | Наличие инстанса (`1`) |

---

### tlsCertificateCollector

Мониторинг срока действия TLS-сертификатов в PEM-файлах на локальной файловой системе.

Поддерживает:
- указание конкретных файлов (`files`);
- рекурсивный обход директорий (`directories`) — все файлы проверяются на наличие PEM-блоков `CERTIFICATE`;
- fullchain-файлы с несколькими сертификатами — каждый сертификат учитывается отдельно;
- исключение CA-сертификатов (`excludeCA: true`) — экспортируются только leaf (серверные/клиентские) сертификаты.

**Интервал обновления:** 600 секунд. Устаревшие метрики (сертификат удалён или заменён) автоматически снимаются при следующем сканировании.

**Конфиг:**

| Параметр | Тип | Описание |
|---|---|---|
| `enabled` | bool | Включить коллектор |
| `excludeCA` | bool | Игнорировать CA-сертификаты (`IsCA=true`) |
| `directories` | []string | Директории для рекурсивного сканирования |
| `files` | []string | Конкретные файлы с сертификатами |

**Пример:**

```yaml
tlsCertificateCollector:
  enabled: true
  excludeCA: true
  directories:
    - /etc/letsencrypt/live/example.com
  files:
    - /etc/ssl/certs/custom.pem
```

**Метрики:**

| Метрика | Тип | Лейблы | Описание |
|---|---|---|---|
| `file_tls_certificate_expiration_seconds` | Gauge | `file`, `common_name`, `issuer`, `dns_names` | Секунды до истечения сертификата (`NotAfter − now`). Отрицательное значение — сертификат просрочен |

Лейблы:

| Лейбл | Описание |
|---|---|
| `file` | Путь к файлу, из которого прочитан сертификат |
| `common_name` | Common Name субъекта |
| `issuer` | Издатель в формате `C=...,O=...,CN=...` (пустые поля опускаются) |
| `dns_names` | DNS-имена из SAN, через запятую (пустая строка, если нет) |

**Пример PromQL-алерта** (сертификат истекает менее чем через 7 дней):

```promql
file_tls_certificate_expiration_seconds < 7 * 24 * 3600
```

---

## Полный пример конфига

```yaml
fileHashCollector:
  enabled: true
  files:
    - /etc/nginx/nginx.conf

portCollector:
  enabled: true
  targets:
    - host: 127.0.0.1
      port: 22
      protocol: TCP

processCollector:
  enabled: true
  processes:
    - process: nginx
      regex: "^nginx$"

systemCollector:
  enabled: true

puppetCollector:
  enabled: false

hetznerCollector:
  enabled: false

hetznerCloudCollector:
  enabled: false

yandexCloudCollector:
  enabled: false

awsCloudCollector:
  enabled: false

tlsCertificateCollector:
  enabled: true
  excludeCA: true
  directories:
    - /etc/letsencrypt/live
  files: []
```
