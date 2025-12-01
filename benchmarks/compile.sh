#!/usr/bin/env bash

set -e

# Create bin directory if it doesn't exist
mkdir -p bin

# Build shadowfax benchmark
echo "Building shadowfax benchmark..."
go build -o ./bin/shadowfax ./benchmarks/shadowfax/main.go

# Build stdlib benchmark
echo "Building stdlib benchmark..."
go build -o ./bin/stdlib ./benchmarks/stdlib/main.go

echo "✓ All benchmarks compiled successfully!"
echo "Binaries available in bin/ directory:"
ls -lh bin/
