#!/usr/bin/env bash
# Run the bot under gdb (Ubuntu 24) so native ntgcalls crashes produce a backtrace.
# Everything is shown in the terminal AND saved to error_log.txt (gdb.txt is also written by gdb).
# Usage: ./run_debug.sh [path-to-binary]     (default: ./main)
set -u
cd "$(dirname "$0")"
BIN="${1:-./main}"
LOG="error_log.txt"

if [ ! -x "$BIN" ]; then
  echo "Binary $BIN not found. Build first:  go generate . && go run setup_ntgcalls.go && CGO_ENABLED=1 go build -o main ." >&2
  exit 1
fi
if ! command -v gdb >/dev/null 2>&1; then
  echo "gdb missing. Install: sudo apt update && sudo apt install -y gdb" >&2
  exit 1
fi

[ -f .env ] && { set -a; . ./.env; set +a; }

export NTG_DEBUG="${NTG_DEBUG:-true}"   # verbose ntgcalls logs
export GOTRACEBACK=crash                # full Go stacks on fatal errors
ulimit -c unlimited 2>/dev/null || true

echo "===== run started $(date -Is) (NTg debug=$NTG_DEBUG) =====" | tee -a "$LOG"

gdb -q -batch \
  -iex "set confirm off" \
  -iex "set pagination off" \
  -iex "handle SIGPIPE pass nostop noprint" \
  -iex "handle SIGINT pass nostop noprint" \
  -iex "handle SIGURG pass nostop noprint" \
  -iex "set logging enabled on" \
  -iex "set debuginfod enabled on" \
  -ex run -ex "thread apply all bt" -ex "info sharedlibrary" \
  --args "$BIN" 2>&1 | tee -a "$LOG"

echo "===== run ended $(date -Is) =====" | tee -a "$LOG"
