# 🦉 SOVA Protocol v1.0.0

```
         ▄▄▄████▄▄▄
       ▄██▀▀    ▀▀██▄      S O V A   N e t w o r k
      ███  ◉    ◉  ███     SOVA Proxy · SOVA Protocol · SOVA VPN
      ███    ▾▾    ███     Autonomous · Stealth · Post-Quantum
       ▀██▄▄▄▄▄▄██▀
      ╱╱ ▀████████▀ ╲╲
     ╱╱   ║██████║   ╲╲
    ▕▕    ║██████║    ▏▏
           ║║  ║║
          ▄╩╩▄▄╩╩▄
```

**SOVA** — автономный стек защищённой сети: собственный локальный прокси, собственный зашифрованный relay-протокол и собственный серверный путь без обязательной опоры на внешние прокси-протоколы.

**Текущий публичный baseline — `v1.0.0`:**
- ✅ **SOVA Proxy** — локальный HTTP CONNECT / plain HTTP ingress
- ✅ **SOVA Protocol** — нативный фреймовый transport поверх TLS
- ✅ **SOVA WebSocket Relay** — тот же SOVA protocol через WebSocket
- ✅ **Stealth + AI stack** — fragmentation, spoofing, jitter, adaptive switching
- ✅ **Post-Quantum Security** — Kyber1024 + Dilithium mode5

Репозиторий остаётся на `v1.0.0`, пока следующий публичный релиз не будет действительно сильнее и стабильнее.

[![License: MIT](https://img.shields.io/badge/License-MIT-purple.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)](https://go.dev)
[![GitHub](https://img.shields.io/badge/GitHub-IvanChernykh%2FSOVA-181717.svg)](https://github.com/IvanChernykh/SOVA)

---

## Что входит в baseline v1.0.0

| Компонент | Описание |
|---|---|
| **SOVA Proxy** | Локальный proxy ingress для браузеров и приложений |
| **SOVA Protocol** | Нативные encrypted frames с handshake и ACK flow |
| **SOVA Relay** | Серверный relay с прямым dial к целям |
| **SOVA WebSocket** | Native SOVA relay поверх WebSocket |
| **Stealth Layer** | TLS camouflage, fragmentation, jitter, padding |
| **System Proxy** | HTTP/HTTPS system-proxy integration |
| **Version** | `v1.0.0` — текущий публичный baseline |

---

## Быстрый старт

### Установка (одна команда)

```bash
# Linux / macOS
curl -fsSL https://raw.githubusercontent.com/IvanChernykh/SOVA/main/install.sh | bash
```

```powershell
# Windows (PowerShell от имени администратора)
powershell -ExecutionPolicy Bypass -Command "iwr -useb https://raw.githubusercontent.com/IvanChernykh/SOVA/main/install.ps1 -OutFile install.ps1; .\install.ps1"
```

### Сборка из исходников

```bash
git clone https://github.com/IvanChernykh/SOVA.git
cd SOVA
go mod tidy
go build -o sova ./client/
go build -o sova-server ./server/
```

### Использование клиента

```bash
# Запуск туннеля (создаёт SOVA Proxy на 127.0.0.1:1080)
# При запуске — анимированная фиолетовая сова летит по терминалу
sova

# Подключение через удалённый SOVA сервер (с DPI evasion)
sova connect server.example.com:443

# Конфигурация
sova config                              # Показать конфигурацию
sova config set listen_port 9090         # Изменить порт
sova config set stealth_profile youtube  # Профиль стелса
sova config set tls_fingerprint firefox  # TLS fingerprint
sova config set transport_mode quic      # Режим транспорта
sova config set jitter_ms 100            # Stealth jitter
sova config set dns_upstream 1.1.1.1:53  # DNS upstream
sova config reset                        # Сбросить к дефолтам
sova config json                         # Вывести конфиг в JSON
sova config path                         # Путь к файлу конфигурации

# Управление модулями (15 модулей)
sova features                            # Статус всех модулей
sova enable dns                          # Включить DNS-over-SOVA
sova disable decoy                       # Выключить decoy-трафик
sova enable mesh_network                 # Включить mesh-сеть
sova enable auto_proxy                   # Авто-настройка системного прокси

# Информация
sova status                              # Статус, статистика, системная информация
sova help                                # Справка по всем командам
sova version                             # Версия
```

### Использование сервера

```bash
# Запуск сервера-релея (с SOVA protocol + TLS)
sova-server

# Конфигурация сервера
sova-server config
sova-server config set port 8443
sova-server config set ai_adapter true
```

### Настройка прокси

После запуска `sova`, настройте браузер или систему:

```
SOVA Proxy (HTTP CONNECT) → 127.0.0.1:1080
```

Или используйте curl:

```bash
curl --proxy http://127.0.0.1:1080 https://youtube.com
```

**Системный прокси автоматически настраивается** при включении `auto_proxy` в конфигурации.

**Никаких зависимостей**: статические бинарники, Go/Python/Node.js не нужны.

---

## Архитектура v1.0.0

### Поток трафика

```
[Браузер/Приложение]
    ↓
[SOVA Proxy — local ingress, 127.0.0.1:1080]
    ↓
[SOVA Client — TLS + Stealth Layer]
    ├─ ClientHello Fragmentation
    ├─ SNI Spoofing
    ├─ Timing Jitter
    └─ Random Padding
    ↓
[SOVA Relay Server — native handler]
    ↓
[Internet]
```

### SOVA Wire Protocol

```
Frame Format:
[Nonce:12] [Length:2] [AES-256-GCM(PadLen:1 | Type:1 | Payload:N | Padding:P)]

Handshake:
1. Client sends SOVA magic + version + salt material
2. Server acknowledges and returns server salt
3. Both sides derive a shared session key from PSK and salts
4. Relay frames move only inside the native SOVA framing layer

Frame Types:
- CONNECT (0x01) — establish connection to target
- DATA (0x02) — encrypted payload
- CLOSE (0x03) — close connection
- KEEPALIVE (0x04) — heartbeat
- ACK (0x05) — acknowledgement
```

### DPI Evasion Techniques

| Техника | Описание | Эффект |
|---|---|---|
| **ClientHello Fragmentation** | Разбиение TLS ClientHello на 2-byte TCP segments | Обход DPI мобильных операторов |
| **SNI Spoofing** | Случайный выбор легитимного домена (google, cloudflare, youtube) | Выглядит как обычный HTTPS трафик |
| **Timing Jitter** | Случайные задержки между фрагментами | Скрывает паттерны отправки |
| **Random Padding** | 4-64 байта padding в каждом фрейме | Скрывает размер реального трафика |
| **Self-Signed Certs** | Генерация сертификатов на лету | Не требует инфраструктуры PKI |

### Безопасность

- **SOVA Wire Protocol**: собственный encrypted frame protocol (Magic + Version + Salt handshake)
- **Шифрование**: AES-256-GCM + ChaCha20-Poly1305 (фреймы + payload)
- **Пост-квантовое**: Kyber1024 (KEM) + Dilithium mode5 (подписи)
- **Аутентификация**: Zero-Knowledge Proof на Ed25519
- **DPI Evasion**: TLS ClientHello фрагментация, SNI spoofing, timing jitter, random padding
- **SOVA Proxy**: собственный локальный ingress для приложений и браузеров
- **DNS-over-SOVA**: Защищённый DNS с кэшированием и fallback

### Ускорение трафика

- **Gzip-сжатие** для уменьшения объёма данных
- **Connection pooling** — переиспользование соединений
- **Route optimizer** — выбор маршрута с минимальной задержкой
- **Hysteria-like CC** — максимальная скорость даже на нестабильных каналах

### Невидимость (Stealth Engine)

- **Traffic mimicry** — имитация профилей Chrome, YouTube, Cloud API
- **Adaptive jitter** — нормальное распределение задержек (Box-Muller)
- **Intelligent padding** — дополнение пакетов до типичных HTTP-размеров
- **Decoy traffic** — фоновые пакеты-обманки
- **TLS fingerprint masking** — маскировка под Chrome, Firefox, Safari

### Offline-First

- Mesh-сеть между устройствами SOVA
- Локальное кэширование
- Bluetooth/NFC peer discovery
- Управление ресурсами батареи

---

## Management API (18 эндпоинтов)

REST API запускается на `http://127.0.0.1:8080/api/` с поддержкой **CORS** и **авторизации по API-ключу** (`X-API-Key` header или `?api_key=` query param).

### Все эндпоинты

| Эндпоинт | Метод | Описание |
|---|---|---|
| `/api/` | GET | Список всех доступных API |
| `/api/status` | GET | Статус системы (uptime, connections, traffic, version) |
| `/api/health` | GET | Health check |
| `/api/config` | GET | Текущая конфигурация (JSON) |
| `/api/config` | PUT | Обновить всю конфигурацию |
| `/api/config/set` | POST | Установить одно значение `{"key": "...", "value": "..."}` |
| `/api/config/reset` | POST | Сбросить конфигурацию к дефолтам |
| `/api/features` | GET | Статус всех 15 модулей (on/off) |
| `/api/feature/` | POST | Включить/выключить модуль `{"name": "...", "enabled": true}` |
| `/api/system` | GET | Системная информация (CPU, RAM, GC, goroutines) |
| `/api/stats` | GET | Статистика трафика (connections, bytes up/down) |
| `/api/logs` | GET | Последние лог-записи (`?limit=N`, по умолчанию 50) |
| `/api/profiles` | GET | Список сохранённых профилей конфигурации |
| `/api/profile` | POST | Переключить профиль `{"name": "..."}` |
| `/api/profile/save` | POST | Сохранить текущий конфиг как профиль `{"name": "..."}` |
| `/api/restart` | POST | Запланировать рестарт туннеля |
| `/api/transport` | GET | Информация о транспорте (mode, SNI, CDN, fallback) |
| `/api/encryption` | GET | Информация о шифровании (algorithm, PQ, ZKP, ciphers) |
| `/api/stealth` | GET | Информация о стелс-движке (profile, jitter, padding, decoy) |

---

## Конфигурация

Конфиг хранится в `~/.sova/config.json`. Все параметры настраиваются через CLI или API.

### Параметры конфигурации

| Ключ | Значения | По умолчанию | Описание |
|---|---|---|---|
| `mode` | `local`, `remote`, `server` | `local` | Режим работы |
| `listen_addr` | IP | `127.0.0.1` | Адрес SOVA Proxy |
| `listen_port` | 1-65535 | `1080` | Порт SOVA Proxy |
| `server_addr` | IP/hostname | — | Адрес удалённого сервера |
| `server_port` | 1-65535 | `443` | Порт сервера |
| `encryption` | `aes-256-gcm`, `chacha20-poly1305` | `aes-256-gcm` | Алгоритм шифрования |
| `stealth_profile` | `chrome`, `youtube`, `cloud_api`, `random` | `chrome` | Профиль стелса |
| `tls_fingerprint` | `chrome`, `firefox`, `safari`, `random` | `chrome` | TLS fingerprint |
| `transport_mode` | `auto`, `web_mirror`, `quic`, `websocket` | `auto` | Режим транспорта |
| `log_level` | `debug`, `info`, `warn`, `error` | `info` | Уровень логирования |
| `api_port` | 1-65535 | `8080` | Порт Management API |
| `dns_upstream` | IP:port | `8.8.8.8:53` | Upstream DNS сервер |
| `dns_port` | 1-65535 | `5353` | Порт DNS-over-SOVA |
| `jitter_ms` | ms | `50` | Stealth jitter задержка |
| `psk` | string | `sova-protocol-v1-key-2026` | Pre-shared key для SOVA protocol |

### Переключаемые модули (15)

```
pq_crypto, zkp, stealth, padding, decoy, ai_adapter,
compression, connection_pool, smart_routing, mesh_network,
offline_first, dns, api, dashboard, auto_proxy
```

```bash
sova enable pq_crypto    # Включить модуль
sova disable decoy       # Выключить модуль
sova features            # Показать статус всех модулей
```

---

## Web Dashboard

Панель управления доступна по адресу `http://localhost:8080` после запуска сервера.

- Статистика в реальном времени
- Активные соединения
- Логи
- Фиолетовая тема с совой

---

## Интеграция

SOVA рассчитан в первую очередь на **нативное внедрение** и на использование через собственные точки входа:

- **Любое приложение** — через встроенный `SOVA Proxy`
- **Сторонние разработчики** — через `sova://` share link и `SOVA profile`
- **Серверные развёртывания** — через нативный SOVA relay over TLS / WebSocket

### Для разработчиков

Если вы хотите встраивать SOVA в сторонние продукты, рекомендуемый путь такой:

1. использовать `SOVA profile` как базовый формат конфигурации;
2. реализовать нативный handshake и frame transport из `common/protocol.go`;
3. поддержать `sova://` share link в UI/импорте конфигов;
4. использовать локальный `SOVA Proxy`, если пока не готова нативная интеграция.

---

## Тестирование

```bash
# Все тесты
go test -v ./common/

# Только криптография
go test -v -run TestEncrypt ./common/

# Только протокол
go test -v -run TestSOVAProtocol ./common/

# Бенчмарки
go test -bench=. -benchmem ./common/
```

58+ тестов покрывают: криптографию, PQ-алгоритмы, AI-адаптацию, сжатие, stealth, аутентификацию, SOVA protocol.

---

## Структура проекта

```
SOVA/
├── server/                  # Сервер
│   ├── main.go                  # Точка входа, CLI, запуск
│   ├── relay.go                 # SOVA relay (TLS + WebSocket transports)
│   ├── api.go                   # Серверный REST API + native SOVA profile export
│   ├── dashboard.go             # Web-дашборд (фиолетовая тема)
│   └── middleware.go            # Rate limiter, logger, connection monitor
├── client/                  # Клиент CLI
│   └── main.go                  # Авто-туннель, SOVA Proxy, 18 команд
├── common/                  # Общая библиотека
│   ├── sova_proxy.go            # SOVA Proxy (HTTP CONNECT / plain HTTP ingress)
│   ├── protocol.go              # SOVA Wire Protocol (frames, encryption, handshake)
│   ├── dpi.go                   # DPI Evasion (fragmentation, SNI spoofing, jitter)
│   ├── config.go                # Конфигурация (профили, JSON, 15 модулей)
│   ├── management_api.go        # REST API управления (18 эндпоинтов, CORS, auth)
│   ├── ui.go                    # Летающая фиолетовая сова + терминальный UI
│   ├── crypto.go                # AES-GCM, ChaCha20, Kyber1024, Dilithium
│   ├── auth.go                  # ZKP аутентификация
│   ├── transport.go             # Транспортные режимы (WebMirror, QUIC, WS)
│   ├── ai.go                    # AI-адаптивное переключение
│   ├── accelerator.go           # Ускорение трафика (gzip, pooling)
│   ├── stealth.go               # Стелс-движок (mimicry, jitter, padding, decoy)
│   ├── upstream.go              # Upstream SOVA gateway / HTTP chaining
│   ├── dns.go                   # DNS-over-SOVA резолвер
│   ├── mesh.go                  # Mesh-сеть между нодами
│   ├── offline_first.go         # Offline-first архитектура
│   └── *_test.go                # Тесты (58+)
├── plugin/                  # Резерв под будущие native integration adapters
├── singbox-patch/           # Reference patch notes and external build materials
├── .github/workflows/       # CI/CD (автосборка + релиз)
├── install.sh               # Установщик Linux/macOS
├── install.ps1              # Установщик Windows
├── Makefile                 # Система сборки (12 целей)
└── go.mod                   # Go модули
```

---

## Поддержка

**SOVA — полностью бесплатный проект. Мы не берём денег.**

- **GitHub Issues**: https://github.com/IvanChernykh/SOVA/issues
- **Discussions**: https://github.com/IvanChernykh/SOVA/discussions
- **Безопасность**: см. [SECURITY.md](SECURITY.md)

---

## Лицензия

MIT License — используйте свободно.

---

*SOVA — тихая, быстрая, невидимая. Как сова в ночи.* 🦉
