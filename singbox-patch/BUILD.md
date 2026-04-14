# SOVA sing-box Build Guide

## Что это

Патч для sing-box, добавляющий нативный протокол SOVA (`"type": "sova"`).
SOVA — полностью автономный зашифрованный VPN-протокол с DPI evasion.

## Архитектура

```
Hiddify (sing-box с SOVA) → TLS(SNI spoof + frag) → SOVA handshake → AES-256-GCM frames → SOVA Server → Internet
```

## Быстрая сборка (Linux)

```bash
cd /root/SOVA/singbox-patch
chmod +x build.sh
./build.sh
```

Результат: `sing-box-sova` бинарник с поддержкой `type=sova`.

## Ручная сборка

### 1. Клонировать sing-box

```bash
git clone --depth 1 --branch v1.8.14 https://github.com/SagerNet/sing-box.git
cd sing-box
```

### 2. Скопировать файлы SOVA

```bash
cp /path/to/singbox-patch/outbound_sova.go outbound/sova.go
cp /path/to/singbox-patch/option_sova.go option/sova.go
```

### 3. Добавить константу

В `constant/type.go`, в блок `const (...)`:
```go
TypeSOVA = "sova"
```

### 4. Зарегистрировать outbound

В `outbound/build.go` (или файл с switch по типам), добавить case:
```go
case C.TypeSOVA:
    return NewSOVA(ctx, router, logger, tag, options.SOVAOptions)
```

### 5. Зарегистрировать option

В `option/outbound.go`, в структуру `Outbound`, добавить поле:
```go
SOVAOptions SOVAOutboundOptions `json:"-"`
```

В `UnmarshalJSON`, добавить case:
```go
case C.TypeSOVA:
    v.SOVAOptions = SOVAOutboundOptions{}
    err = UnmarshalJSON(content, &v.SOVAOptions)
```

### 6. Собрать

```bash
go mod tidy
CGO_ENABLED=0 go build -tags "with_quic,with_utls" -o sing-box-sova ./cmd/sing-box
```

## Конфигурация sing-box

```json
{
  "outbounds": [{
    "type": "sova",
    "tag": "CUPOL-SOVA",
    "server": "cupol.space",
    "server_port": 9443,
    "psk": "sova-protocol-v1-key-2026",
    "sni_list": ["www.google.com", "cdn.cloudflare.com"],
    "fragment_size": 2,
    "fragment_jitter": 25
  }]
}
```

## SOVA Share Link

```
sova://cupol.space:9443?psk=sova-protocol-v1-key-2026&frag=2&jitter=25#CUPOL-SOVA
```

## Сборка для Android (Hiddify)

```bash
./build.sh --android
```

Заменить sing-box binary в Hiddify APK:
1. Распаковать Hiddify APK
2. Заменить `lib/arm64-v8a/libcore.so` на собранный `sing-box-sova-android-arm64`
3. Пересобрать и подписать APK

## Серверная часть

SOVA сервер запускается отдельно:
```bash
SOVA_UUID=... /usr/local/bin/sova-server
```

Порт 9443 — нативный SOVA протокол (TLS + SOVA handshake + encrypted frames).
