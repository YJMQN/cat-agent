#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_DIR="$ROOT_DIR/.devlogs"
BACKEND_LOG="$LOG_DIR/backend.log"
FRONTEND_LOG="$LOG_DIR/frontend.log"

mkdir -p "$LOG_DIR"

cleanup() {
  if [[ -n "${BACKEND_PID:-}" ]] && kill -0 "$BACKEND_PID" 2>/dev/null; then
    kill "$BACKEND_PID" 2>/dev/null || true
    wait "$BACKEND_PID" 2>/dev/null || true
  fi

  if [[ -n "${FRONTEND_PID:-}" ]] && kill -0 "$FRONTEND_PID" 2>/dev/null; then
    kill "$FRONTEND_PID" 2>/dev/null || true
    wait "$FRONTEND_PID" 2>/dev/null || true
  fi

  rm -f "$ROOT_DIR/server"
}

trap cleanup EXIT INT TERM

if curl -fsS http://localhost:8080/health >/dev/null 2>&1; then
  echo "后端已在运行：http://localhost:8080"
else
  echo "启动后端..."
  cd "$ROOT_DIR"
  go build -o server ./cmd/main.go
  ./server >"$BACKEND_LOG" 2>&1 &
  BACKEND_PID=$!

  for _ in {1..30}; do
    if curl -fsS http://localhost:8080/health >/dev/null 2>&1; then
      break
    fi
    sleep 1
  done

  if ! curl -fsS http://localhost:8080/health >/dev/null 2>&1; then
    echo "后端启动失败，请查看日志：$BACKEND_LOG"
    exit 1
  fi

  echo "后端已启动：http://localhost:8080"
fi

if curl -fsS http://localhost:3000 >/dev/null 2>&1; then
  echo "前端已在运行：http://localhost:3000"
else
  echo "启动前端..."
  cd "$ROOT_DIR/web"
  npm run dev -- --host 0.0.0.0 --port 3000 >"$FRONTEND_LOG" 2>&1 &
  FRONTEND_PID=$!

  for _ in {1..30}; do
    if curl -fsS http://localhost:3000 >/dev/null 2>&1; then
      break
    fi
    sleep 1
  done

  if ! curl -fsS http://localhost:3000 >/dev/null 2>&1; then
    echo "前端启动失败，请查看日志：$FRONTEND_LOG"
    exit 1
  fi

  echo "前端已启动：http://localhost:3000"
fi

echo "日志目录：$LOG_DIR"
echo "按 Ctrl+C 停止服务"

if [[ -n "${BACKEND_PID:-}" || -n "${FRONTEND_PID:-}" ]]; then
  wait
fi
