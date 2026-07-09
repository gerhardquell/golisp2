#!/bin/bash
# build.sh – kompiliert alle GoLisp-Binaries nach ./build/
#   golisp        (CLI/REPL/SWANK)
#   golispd       (TCP-Server)
#   golisp-client (Client/REPL)
#
# CLAUDE.md: "verwende ./build für die Builds" – ./build/ ist das Output-Verz.
set -e

cd "$(dirname "$0")"
mkdir -p build

echo "→ go vet"
go vet ./lib/ ./lib/swank/ 2>/dev/null || true

echo "→ build golisp"
go build -o build/golisp .

echo "→ build golispd"
go build -o build/golispd ./cmd/golispd/

echo "→ build golisp-client"
go build -o build/golisp-client ./cmd/golisp-client/

echo "✓ Binaries in ./build/:"
ls -1 build/
