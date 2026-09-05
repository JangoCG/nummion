#!/usr/bin/env bash
set -euo pipefail
mkdir -p completions
go run ./cmd/num completion bash > completions/num.bash
go run ./cmd/num completion zsh > completions/_num
go run ./cmd/num completion fish > completions/num.fish
go run ./cmd/num completion powershell > completions/num.ps1
