#!/bin/bash
set -euo pipefail

KEYSTORE_OUT="client/gift/src-tauri/gen/android/app/gift.keystore"
PROPS_OUT="client/gift/src-tauri/gen/android/keystore.properties"

missing=()
for var in KEYSTORE_BASE64 KEYSTORE_PASSWORD KEY_ALIAS KEY_PASSWORD; do
  if [ -z "${!var:-}" ]; then
    missing+=("$var")
  fi
done
if [ ${#missing[@]} -gt 0 ]; then
  echo "ERROR: required env vars not set: ${missing[*]}" >&2
  echo "Export them before running goreleaser (KEYSTORE_BASE64 is the base64-encoded .keystore file)." >&2
  exit 1
fi

printf '%s' "$KEYSTORE_BASE64" | base64 --decode > "$KEYSTORE_OUT"

if [ ! -s "$KEYSTORE_OUT" ]; then
  echo "ERROR: decoded keystore at $KEYSTORE_OUT is empty — KEYSTORE_BASE64 is not valid base64." >&2
  exit 1
fi

if command -v keytool >/dev/null 2>&1; then
  if ! keytool -list \
      -keystore "$KEYSTORE_OUT" \
      -storepass "$KEYSTORE_PASSWORD" \
      -alias "$KEY_ALIAS" >/dev/null 2>&1; then
    echo "ERROR: keystore at $KEYSTORE_OUT cannot be read with the provided password/alias." >&2
    exit 1
  fi
fi

cat > "$PROPS_OUT" << EOF
storeFile=gift.keystore
storePassword=${KEYSTORE_PASSWORD}
keyAlias=${KEY_ALIAS}
keyPassword=${KEY_PASSWORD}
EOF

echo "✓ Keystore decoded to $KEYSTORE_OUT ($(wc -c < "$KEYSTORE_OUT") bytes)"
echo "✓ keystore.properties written"
