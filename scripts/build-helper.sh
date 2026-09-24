#!/bin/bash

set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)

mkdir -p "$ROOT/bin"
(
  cd "$ROOT/backend"
  CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' \
    -o "$ROOT/bin/omarchy-musicfox-helper" ./cmd/omarchy-musicfox-helper
)
chmod 0755 "$ROOT/bin/omarchy-musicfox-helper"

echo "Built bin/omarchy-musicfox-helper"
