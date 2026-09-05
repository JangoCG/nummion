#!/usr/bin/env bash
set -euo pipefail
mkdir -p completions
go run ./cmd/nummion completion bash > completions/num.bash
go run ./cmd/nummion completion zsh > completions/_num
go run ./cmd/nummion completion fish > completions/num.fish
go run ./cmd/nummion completion powershell > completions/num.ps1
