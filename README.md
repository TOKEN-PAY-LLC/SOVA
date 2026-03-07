# SOVA Protocol v2.0

```
    ,___,
    {o,o}    Secure Obfuscated Versatile Adapter
    /)  )    AI-Powered | Post-Quantum | Free & Open Source
    -"  "-
```

**SOVA** — бесплатный протокол нового поколения для защищённой передачи данных. Ускоряет интернет, обходит DPI, невидим как сова в ночи.

[![License: MIT](https://img.shields.io/badge/License-MIT-purple.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)](https://go.dev)
[![GitHub](https://img.shields.io/badge/GitHub-IvanChernykh%2FSOVA-181717.svg)](https://github.com/IvanChernykh/SOVA)

---

## Почему SOVA?

| Возможность | Описание |
|---|---|
| **Ускорение трафика** | Сжатие gzip, пулинг соединений, умная маршрутизация |
| **Невидимость** | Мимикрия под Chrome/YouTube/CloudAPI, адаптивный jitter, decoy-трафик |
| **AI-адаптация** | Автоматический обход DPI без участия пользователя |
| **Пост-квантовая криптография** | Kyber1024 KEM + Dilithium5 подписи (Cloudflare circl) |
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

Установщик автоматически определяет ОС и архитектуру, скачивает бинарник из [GitHub Releases](https://github.com/IvanChernykh/SOVA/releases/latest), настраивает сервис и конфигурацию.

### Сборка из исходников

```bash
git clone https://github.com/IvanChernykh/SOVA.git
cd SOVA
go mod tidy
go build -o bin/sova-server ./server/
go build -o bin/sova ./client/
```

### Использование

```bash
# Запуск сервера
sova-server

# Подключение клиента
sova connect <base64_config>

# Статус
sova status

# Web-панель управления
# Открыть http://localhost:8080
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

## API

REST API доступен по `/api/*`:

| Эндпоинт | Метод | Описание |
|---|---|---|
| `/api/stats` | GET | Статистика сервера |
| `/api/register` | POST | Регистрация пользователя |
| `/api/config?user_id=<id>` | GET | Получить конфигурацию |
| `/api/export?client=<xray\|singbox>&user_id=<id>` | GET | Экспорт для прокси-клиентов |
| `/api/proxy` | GET | Ссылки для прокси-клиентов |

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
├── server/           # Сервер (main.go, api.go, dashboard.go, config.go, middleware.go)
├── client/           # Клиент CLI (main.go)
├── common/           # Общая библиотека
│   ├── crypto.go         # AES-GCM, ChaCha20, Kyber1024, Dilithium
│   ├── auth.go           # ZKP аутентификация
│   ├── transport.go      # Транспортные режимы
│   ├── ai.go             # AI-адаптер
│   ├── accelerator.go    # Ускорение трафика
│   ├── stealth.go        # Движок скрытности
│   ├── socks5.go         # SOCKS5 прокси
│   ├── dns.go            # DNS-over-SOVA
│   ├── ui.go             # Терминальный UI с совой
│   └── *_test.go         # Тесты
├── plugin/           # Плагины (Xray, Sing-Box)
├── install.sh        # Установщик Linux/macOS
├── install.ps1       # Установщик Windows
├── Makefile          # Система сборки
└── go.mod            # Go модули
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