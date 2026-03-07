# SOVA Protocol v1.0.0

```
        ▄▄▄▄▄▄▄▄▄▄▄
       ▐  ◉      ◉  ▌    Secure Obfuscated Versatile Adapter
       ▐     ▼▼     ▌    AI-Powered | Post-Quantum | Free & Open Source
        ▀▄▄▄▄▄▄▄▄▀
         ╱╱    ╲╲
        ╱╱  ██  ╲╲
       ▕▕   ██   ▏▏
```

**SOVA** — бесплатный протокол нового поколения для защищённой передачи данных. Работает как VPN, обходит DPI, невидим как сова в ночи.

[![License: MIT](https://img.shields.io/badge/License-MIT-purple.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)](https://go.dev)
[![GitHub](https://img.shields.io/badge/GitHub-IvanChernykh%2FSOVA-181717.svg)](https://github.com/IvanChernykh/SOVA)

---

## Почему SOVA?

| Возможность | Описание |
|---|---|
| **Работает из коробки** | Запустил — туннель создан — YouTube открывается |
| **Ускорение трафика** | Сжатие gzip, пулинг соединений, умная маршрутизация |
| **Невидимость** | Мимикрия под Chrome/YouTube/CloudAPI, адаптивный jitter, decoy-трафик |
| **AI-адаптация** | Автоматический обход DPI без участия пользователя |
| **Пост-квантовая криптография** | Kyber1024 KEM + Dilithium5 подписи (Cloudflare circl) |
| **Гибкие API** | REST API для управления: включение/выключение модулей, конфигурация |
| **Бесплатно и навсегда** | MIT лицензия, никаких платных планов |

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
# Запуск туннеля (создаёт SOCKS5 прокси на 127.0.0.1:1080)
sova

# Подключение через удалённый SOVA сервер
sova connect server.example.com:443

# Конфигурация
sova config                          # Показать конфигурацию
sova config set listen_port 9090     # Изменить порт
sova config set stealth_profile youtube  # Профиль стелса
sova config reset                    # Сбросить к дефолтам
sova config json                     # Вывести конфиг в JSON

# Управление модулями
sova features                        # Статус всех модулей
sova enable dns                      # Включить DNS-over-SOVA
sova disable decoy                   # Выключить decoy-трафик
sova enable mesh_network             # Включить mesh-сеть

# Информация
sova status                          # Статус и системная информация
sova help                            # Справка
sova version                         # Версия
```

### Использование сервера

```bash
# Запуск сервера-релея
sova-server

# Конфигурация сервера
sova-server config
sova-server config set port 8443
sova-server config set ai_adapter true
```

### Настройка прокси

После запуска `sova`, настройте браузер или систему:

```
SOCKS5 → 127.0.0.1:1080
```

Или используйте curl:

```bash
curl --proxy socks5h://127.0.0.1:1080 https://youtube.com
```

**Никаких зависимостей**: статические бинарники, Go/Python/Node.js не нужны.

---

## Архитектура

### Транспортные режимы

| Режим | Порт | Технология | Когда использовать |
|---|---|---|---|
| **Web Mirror** | TCP 443 | Custom TLS handshake | По умолчанию, имитация HTTPS |
| **Cloud Carrier** | UDP | QUIC + adaptive CC | Высокая скорость, потери |
| **Shadow WebSocket** | TCP 443 | WebSocket через CDN | Жёсткая блокировка |

Переключение между режимами — **автоматическое** на основе AI-анализа сети.

### Безопасность

- **Шифрование**: AES-256-GCM + ChaCha20-Poly1305
- **Пост-квантовое**: Kyber1024 (KEM) + Dilithium mode5 (подписи)
- **Аутентификация**: Zero-Knowledge Proof на Ed25519
- **Обфускация**: Packet morphing, timing jitter, SNI rotation, TLS fingerprint mimicry
- **SOCKS5 прокси**: Встроенный для совместимости с любыми приложениями
- **DNS-over-SOVA**: Защищённый DNS с кэшированием

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
- **TLS fingerprint masking** — маскировка под популярные браузеры

### Offline-First

- Mesh-сеть между устройствами SOVA
- Локальное кэширование
- Bluetooth/NFC peer discovery
- Управление ресурсами батареи

---

## Web Dashboard

Панель управления доступна по адресу `http://localhost:8080` после запуска сервера.

- Статистика в реальном времени
- Активные соединения
- Логи
- Фиолетовая тема с совой

---

## Management API

REST API автоматически запускается на `http://127.0.0.1:8080/api/`:

### Клиентский API

| Эндпоинт | Метод | Описание |
|---|---|---|
| `/api/status` | GET | Статус системы (uptime, mode, version) |
| `/api/health` | GET | Health check |
| `/api/config` | GET | Текущая конфигурация |
| `/api/config` | PUT | Обновить всю конфигурацию |
| `/api/config/set` | POST | Установить одно значение `{"key": "...", "value": "..."}` |
| `/api/features` | GET | Статус всех модулей |
| `/api/feature/` | POST | Включить/выключить модуль `{"name": "...", "enabled": true}` |
| `/api/system` | GET | Системная информация (CPU, RAM, goroutines) |

### Серверный API (дополнительно)

| Эндпоинт | Метод | Описание |
|---|---|---|
| `/api/register` | POST | Регистрация пользователя |
| `/api/stats` | GET | Статистика сервера |
| `/api/export?client=xray&user_id=...` | GET | Экспорт конфига для Xray/SingBox/Clash |
| `/api/proxy` | GET | Ссылки для прокси-клиентов |

### Примеры

```bash
# Статус
curl http://127.0.0.1:8080/api/status

# Включить DNS
curl -X POST http://127.0.0.1:8080/api/feature/ -d '{"name":"dns","enabled":true}'

# Изменить порт
curl -X POST http://127.0.0.1:8080/api/config/set -d '{"key":"listen_port","value":"9090"}'

# Полная конфигурация в JSON
curl http://127.0.0.1:8080/api/config
```

---

## Совместимость

SOVA работает как плагин для:
- **Xray / V2Ray** — расширение VLESS+XTLS
- **Sing-Box** — нативный outbound
- **Любое приложение** — через встроенный SOCKS5 прокси

---

## Тестирование

```bash
# Все тесты
go test -v ./common/

# Только криптография
go test -v -run TestEncrypt ./common/

# Бенчмарки
go test -bench=. -benchmem ./common/
```

44+ тестов покрывают: криптографию, PQ-алгоритмы, AI-адаптацию, сжатие, stealth, аутентификацию.

---

## Структура проекта

```
SOVA/
├── server/               # Сервер
│   ├── main.go               # Точка входа, CLI, запуск
│   ├── relay.go              # Релей трафика (клиент→интернет)
│   ├── api.go                # Серверный REST API + регистрация
│   ├── dashboard.go          # Web-дашборд (фиолетовая тема)
│   └── middleware.go         # Rate limiter, logger, connection monitor
├── client/               # Клиент CLI
│   └── main.go               # Авто-туннель, SOCKS5, команды
├── common/               # Общая библиотека
│   ├── config.go             # Система конфигурации (профили, JSON)
│   ├── management_api.go     # REST API управления (enable/disable, CRUD)
│   ├── crypto.go             # AES-GCM, ChaCha20, Kyber1024, Dilithium
│   ├── auth.go               # ZKP аутентификация
│   ├── transport.go          # Транспортные режимы (WebMirror, QUIC, WS)
│   ├── ai.go                 # AI-адаптивное переключение
│   ├── accelerator.go        # Ускорение трафика (gzip, pooling)
│   ├── stealth.go            # Движок скрытности (mimicry, jitter)
│   ├── socks5.go             # SOCKS5 прокси-сервер
│   ├── dns.go                # DNS-over-SOVA резолвер
│   ├── mesh.go               # Mesh-сеть между нодами
│   ├── offline_first.go      # Offline-first архитектура
│   ├── ui.go                 # Анимированная сова + терминальный UI
│   └── *_test.go             # Тесты (58+)
├── plugin/               # Плагины (Xray, Sing-Box)
├── install.sh            # Установщик Linux/macOS
├── install.ps1           # Установщик Windows
├── Makefile              # Система сборки
└── go.mod                # Go модули
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

*SOVA — тихая, быстрая, невидимая. Как сова в ночи.*