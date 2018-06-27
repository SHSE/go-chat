#!/usr/bin/env bash
GOMAXPROCS=2 CHAT_PORT=3000 ./build/chat &
PID=$!
trap "kill ${PID}" EXIT
SERVER_URL=localhost:3000 vgo test -bench=. -run=none ./transport
