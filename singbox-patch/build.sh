#!/bin/bash
# ══════════════════════════════════════════════════════════════════════════
# SOVA sing-box Builder — собирает sing-box с нативным SOVA протоколом
# ══════════════════════════════════════════════════════════════════════════
#
# Использование:
#   chmod +x build.sh
#   ./build.sh
#
# Результат:
#   ./sing-box-sova (бинарник sing-box с поддержкой type=sova)
#
# ══════════════════════════════════════════════════════════════════════════

set -e

SINGBOX_REPO="https://github.com/SagerNet/sing-box.git"
SINGBOX_VERSION="v1.8.14"  # Hiddify-совместимая версия
PATCH_DIR="$(cd "$(dirname "$0")" && pwd)"
BUILD_DIR="/tmp/singbox-sova-build"
OUTPUT="$PATCH_DIR/sing-box-sova"

echo "🦉 SOVA sing-box Builder"
echo "   sing-box: $SINGBOX_VERSION"
echo "   Patch dir: $PATCH_DIR"
echo ""

# ─── 1. Клонируем sing-box ───
echo ">>> Step 1: Cloning sing-box $SINGBOX_VERSION..."
rm -rf "$BUILD_DIR"
git clone --depth 1 --branch "$SINGBOX_VERSION" "$SINGBOX_REPO" "$BUILD_DIR" 2>/dev/null || {
    echo "Branch $SINGBOX_VERSION not found, trying main..."
    git clone --depth 1 "$SINGBOX_REPO" "$BUILD_DIR"
}
cd "$BUILD_DIR"
echo "    ✅ Cloned to $BUILD_DIR"

# ─── 2. Копируем SOVA файлы ───
echo ">>> Step 2: Patching sing-box with SOVA protocol..."

# Outbound
cp "$PATCH_DIR/outbound_sova.go" outbound/sova.go
echo "    ✅ outbound/sova.go"

# Option
cp "$PATCH_DIR/option_sova.go" option/sova.go
echo "    ✅ option/sova.go"

# ─── 3. Добавляем константу TypeSOVA ───
if ! grep -q 'TypeSOVA' constant/type.go 2>/dev/null; then
    # Добавляем константу в конец блока const
    sed -i '/^const (/,/)/ {
        /^)/ i\\tTypeSOVA = "sova"
    }' constant/type.go || {
        # Если формат другой, добавляем отдельно
        echo 'const TypeSOVA = "sova"' >> constant/type.go
    }
    echo "    ✅ constant/type.go: TypeSOVA added"
else
    echo "    ✅ constant/type.go: TypeSOVA already exists"
fi

# ─── 4. Регистрируем SOVA в outbound builder ───
# Находим файл с switch по outbound типам
BUILDER_FILE=$(grep -rl 'case C.TypeVLESS' outbound/ 2>/dev/null | head -1)
if [ -n "$BUILDER_FILE" ]; then
    if ! grep -q 'TypeSOVA' "$BUILDER_FILE"; then
        # Добавляем case для SOVA перед default
        sed -i "/case C.TypeVLESS/a\\
\\tcase C.TypeSOVA:\\
\\t\\treturn NewSOVA(ctx, router, logger, tag, options.SOVAOptions)" "$BUILDER_FILE"
        echo "    ✅ $BUILDER_FILE: SOVA case added"
    else
        echo "    ✅ $BUILDER_FILE: SOVA already registered"
    fi
else
    echo "    ⚠  Could not find outbound builder, manual patch needed"
fi

# ─── 5. Регистрируем SOVAOptions в option/outbound.go ───
OPTION_FILE="option/outbound.go"
if [ -f "$OPTION_FILE" ]; then
    if ! grep -q 'SOVAOptions' "$OPTION_FILE"; then
        # Добавляем поле SOVAOptions в структуру Outbound
        sed -i '/VLESSOptions/a\\tSOVAOptions         SOVAOutboundOptions `json:"-"`' "$OPTION_FILE"
        echo "    ✅ $OPTION_FILE: SOVAOptions field added"

        # Добавляем case в UnmarshalJSON
        if grep -q 'UnmarshalJSON' "$OPTION_FILE"; then
            sed -i '/case C.TypeVLESS/,/v.VLESSOptions/ {
                /v.VLESSOptions/a\\
\\tcase C.TypeSOVA:\\
\\t\\tv.SOVAOptions = SOVAOutboundOptions{}\\
\\t\\terr = UnmarshalJSON(content, \&v.SOVAOptions)
            }' "$OPTION_FILE" 2>/dev/null || true
        fi
        
        # Добавляем case в MarshalJSON
        if grep -q 'MarshalJSON' "$OPTION_FILE"; then
            sed -i '/case C.TypeVLESS/,/v.VLESSOptions/ {
                /v.VLESSOptions.*MarshalJSON\|MarshalJSON.*v.VLESSOptions/a\\
\\tcase C.TypeSOVA:\\
\\t\\treturn MarshallObjects(v.SOVAOptions)
            }' "$OPTION_FILE" 2>/dev/null || true
        fi
        
        echo "    ✅ $OPTION_FILE: JSON marshaling patched"
    else
        echo "    ✅ $OPTION_FILE: SOVAOptions already exists"
    fi
else
    echo "    ⚠  $OPTION_FILE not found, manual patch needed"
fi

# ─── 6. go mod tidy ───
echo ">>> Step 3: Resolving dependencies..."
go mod tidy
echo "    ✅ Dependencies resolved"

# ─── 7. Build ───
echo ">>> Step 4: Building sing-box with SOVA..."

# Определяем платформу
GOOS=${GOOS:-$(go env GOOS)}
GOARCH=${GOARCH:-$(go env GOARCH)}

CGO_ENABLED=0 go build \
    -tags "with_quic,with_utls" \
    -ldflags "-s -w -X 'github.com/sagernet/sing-box/constant.Version=${SINGBOX_VERSION}-sova'" \
    -o "$OUTPUT" \
    ./cmd/sing-box

echo ""
echo "══════════════════════════════════════════════════════════════════"
echo "  🦉 BUILD COMPLETE"
echo "  Binary: $OUTPUT"
echo "  Size: $(du -h "$OUTPUT" | cut -f1)"
echo "  Platform: $GOOS/$GOARCH"
echo ""
echo "  SOVA protocol: ✅ type=\"sova\" available"
echo ""
echo "  Test: $OUTPUT run -c config.json"
echo "══════════════════════════════════════════════════════════════════"

# ─── 8. Cross-compile для Android (опционально) ───
if [ "$1" == "--android" ]; then
    echo ""
    echo ">>> Building for Android (arm64)..."
    CGO_ENABLED=0 GOOS=android GOARCH=arm64 go build \
        -tags "with_quic,with_utls" \
        -ldflags "-s -w" \
        -o "${OUTPUT}-android-arm64" \
        ./cmd/sing-box
    echo "    ✅ ${OUTPUT}-android-arm64"
fi

echo ""
echo "Done! 🦉"
