#!/bin/bash
echo $KEYSTORE_BASE64 | base64 --decode > client/gift/src-tauri/gen/android/app/gift.keystore
export KEYSTORE_PATH="$PWD/client/gift/src-tauri/gen/android/app/gift.keystore"