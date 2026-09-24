#!/bin/bash

set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
OMARCHY_PATH=${OMARCHY_PATH:-/usr/share/omarchy}
QMLLINT=${QMLLINT:-/usr/lib/qt6/bin/qmllint}

omarchy plugin validate "$ROOT"
node "$ROOT/tests/i18n.test.js"
node "$ROOT/tests/lyrics.test.js"
(
  cd "$ROOT/backend"
  go test ./...
  go vet ./...
)
"$ROOT/scripts/build-helper.sh"
"$QMLLINT" -I "$OMARCHY_PATH/shell" "$ROOT/Service.qml" "$ROOT/BarWidget.qml"

echo "All checks passed"
