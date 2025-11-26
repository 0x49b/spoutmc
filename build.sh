#!/bin/bash

set -e  # Exit on error

echo "======================================="
echo "  SpoutMC Multi-Architecture Build"
echo "======================================="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WEB_DIR="$PROJECT_ROOT/web"
EMBED_DIR="$PROJECT_ROOT/internal/webserver/static/dist"
OUTPUT_DIR="$PROJECT_ROOT/build"
VERSION="0.0.1"

# Build targets
TARGETS=(
    "linux/amd64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

# Step 1: Clean previous builds
echo -e "${YELLOW}[1/6] Cleaning previous builds...${NC}"
rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"
rm -rf "$EMBED_DIR"

# Step 2: Install frontend dependencies
echo -e "${YELLOW}[2/6] Installing frontend dependencies...${NC}"
cd "$WEB_DIR"
if [ ! -d "node_modules" ]; then
    npm install
else
    echo "  → Dependencies already installed"
fi

# Step 3: Build frontend
echo -e "${YELLOW}[3/6] Building frontend (Vite)...${NC}"
npm run build

# Verify build output
if [ ! -d "$WEB_DIR/dist" ]; then
    echo -e "${RED}✗ Frontend build failed - dist directory not found${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Frontend build complete${NC}"

# Step 4: Copy frontend to embed directory
echo -e "${YELLOW}[4/6] Copying frontend to embed directory...${NC}"
mkdir -p "$EMBED_DIR"
cp -r "$WEB_DIR/dist/"* "$EMBED_DIR/"
echo -e "${GREEN}✓ Frontend copied to $EMBED_DIR${NC}"

# Step 5: Generate Swagger docs
echo -e "${YELLOW}[5/6] Generating Swagger documentation...${NC}"
cd "$PROJECT_ROOT"
if command -v swag &> /dev/null; then
    swag init -g cmd/spoutmc/main.go --parseDependency --parseInternal
    echo -e "${GREEN}✓ Swagger docs generated${NC}"
else
    echo -e "${YELLOW}⚠ swag not found, skipping Swagger generation${NC}"
fi

# Step 6: Build Go binaries
echo -e "${YELLOW}[6/6] Building Go binaries for multiple architectures...${NC}"
cd "$PROJECT_ROOT"

for target in "${TARGETS[@]}"; do
    GOOS="${target%/*}"
    GOARCH="${target#*/}"

    OUTPUT_NAME="spoutmc-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        OUTPUT_NAME="${OUTPUT_NAME}.exe"
    fi

    echo -e "  Building for ${GOOS}/${GOARCH}..."

    GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags="-s -w -X main.Version=${VERSION}" \
        -o "$OUTPUT_DIR/$OUTPUT_NAME" \
        ./cmd/spoutmc

    if [ $? -eq 0 ]; then
        FILE_SIZE=$(du -h "$OUTPUT_DIR/$OUTPUT_NAME" | cut -f1)
        echo -e "${GREEN}  ✓ Built $OUTPUT_NAME ($FILE_SIZE)${NC}"
    else
        echo -e "${RED}  ✗ Failed to build for ${GOOS}/${GOARCH}${NC}"
        exit 1
    fi
done

# Summary
echo ""
echo -e "${GREEN}=======================================${NC}"
echo -e "${GREEN}  Build Complete!${NC}"
echo -e "${GREEN}=======================================${NC}"
echo ""
echo "Built binaries:"
ls -lh "$OUTPUT_DIR"
echo ""
echo "To run:"
echo "  Linux:   ./build/spoutmc-linux-amd64"
echo "  macOS:   ./build/spoutmc-darwin-amd64 (or darwin-arm64)"
echo "  Windows: ./build/spoutmc-windows-amd64.exe"
