#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
MODE="${1:-quick}"

exec go run "$ROOT_DIR/scripts/check" "$MODE"