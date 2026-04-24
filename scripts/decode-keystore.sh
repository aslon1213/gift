#!/bin/bash
set -e

KEYSTORE_OUT="client/gift/src-tauri/gen/android/app/gift.keystore"
echo "$KEYSTORE_BASE64" | base64 --decode > "$KEYSTORE_OUT"
echo "Keystore decoded to $KEYSTORE_OUT"

# Export for subsequent hooks
export KEYSTORE_PATH="$(pwd)/$KEYSTORE_OUT"