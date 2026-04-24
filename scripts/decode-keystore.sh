#!/bin/bash
set -e

KEYSTORE_OUT="client/gift/src-tauri/gen/android/app/gift.keystore"
PROPS_OUT="client/gift/src-tauri/gen/android/keystore.properties"

echo "$KEYSTORE_BASE64" | base64 --decode > "$KEYSTORE_OUT"

cat > "$PROPS_OUT" << EOF
storeFile=app/gift.keystore
storePassword=${KEYSTORE_PASSWORD}
keyAlias=${KEY_ALIAS}
keyPassword=${KEY_PASSWORD}
EOF

echo "✓ Keystore decoded"
echo "✓ keystore.properties written"