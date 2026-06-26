#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo_root"

echo "==> gofmt"
gofmt -w .
git diff --exit-code

echo "==> go test"
go test ./...

echo "==> go test -race"
go test -race ./...

echo "==> go vet"
go vet ./...

echo "==> whitespace"
git diff --check

echo "==> Ruby syntax and Formula helper tests"
ruby -c scripts/update_formula.rb
ruby -c Formula/disk-smi.rb
ruby scripts/update_formula_test.rb

echo "==> darwin cross-builds"
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o /tmp/disk-smi-darwin-amd64 ./cmd/disk-smi
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o /tmp/disk-smi-darwin-arm64 ./cmd/disk-smi

echo "release readiness checks passed"
