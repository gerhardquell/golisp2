#!/bin/bash
# build.sh – kompiliert alle GoLisp2-Binaries nach ./build/
#   golisp2         (CLI/REPL/SWANK — SWANK-Server per `golisp2 --swank host:port`)
#   golisp2-client  (Client/REPL)
#
# CLAUDE.md: "verwende ./build für die Builds" – ./build/ ist das Output-Verz.
set -e

cd "$(dirname "$0")"
mkdir -p build

echo "→ go vet"
go vet ./src/lib/ ./src/lib/swank/ 2>/dev/null || true

echo "→ build golisp2"
go build -o build/golisp2 ./src/

echo "→ build golisp2-client"
go build -o build/golisp2-client ./src/cmd/golisp2-client/

echo "✓ Binaries in ./build/:"
ls -1 build/
