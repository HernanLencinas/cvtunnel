#!/bin/bash

set -euo pipefail

# Variables
BUILD_DIR="./build"
BINARY_NAME="cvtun"

# Crear directorio de salida
mkdir -p "$BUILD_DIR"

# Función para realizar el build en una plataforma específica
build_for_platform() {
    local os="$1"
    local arch="$2"
    local output_name="${BINARY_NAME}-${os}_${arch}"
    local output_path="${BUILD_DIR}/${os}_${arch}/${output_name}"

    echo "🔧 Building for ${os}/${arch}..."

    mkdir -p "$(dirname "$output_path")"

    # Compilar
    if env GOOS="$os" GOARCH="$arch" go build -o "$output_path"; then
        echo "✅ Build successful: $output_path"
    else
        echo "❌ Build failed for ${os}/${arch}" >&2
    fi
}

# Lista de plataformas a compilar
platforms=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

# Compilar para cada plataforma
for platform in "${platforms[@]}"; do
    IFS="/" read -r os arch <<< "$platform"
    build_for_platform "$os" "$arch"
done