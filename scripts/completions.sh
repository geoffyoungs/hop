#!/bin/bash
set -e

mkdir -p completions
go run ./cmd/hop completion bash > completions/hop.bash
go run ./cmd/hop completion zsh > completions/hop.zsh
go run ./cmd/hop completion fish > completions/hop.fish
