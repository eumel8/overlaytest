#!/bin/bash
# Test script for overlaytest container image
# Usage: ./test-image.sh [image-name]

IMAGE="${1:-ghcr.io/eumel8/overlaytest:latest}"

echo "Testing container image: $IMAGE"
echo "======================================"
echo

# Test 1: Check if image exists
echo "Test 1: Check if image exists..."
if docker image inspect "$IMAGE" &>/dev/null; then
    echo "✓ Image exists"
else
    echo "✗ Image not found"
    exit 1
fi

# Test 2: Check image size
echo
echo "Test 2: Check image size..."
SIZE=$(docker image inspect "$IMAGE" --format='{{.Size}}' | awk '{print $1/1024/1024}')
echo "Image size: ${SIZE}MB"
if (( $(echo "$SIZE < 50" | bc -l) )); then
    echo "✓ Image is small (<50MB)"
else
    echo "⚠ Image is larger than expected"
fi

# Test 3: Test bash availability
echo
echo "Test 3: Test bash availability..."
if docker run --rm "$IMAGE" bash --version &>/dev/null; then
    echo "✓ Bash is available"
else
    echo "✗ Bash not found"
    exit 1
fi

# Test 4: Test ping command
echo
echo "Test 4: Test ping command..."
if docker run --rm "$IMAGE" ping -c 2 127.0.0.1 &>/dev/null; then
    echo "✓ Ping works"
else
    echo "✗ Ping failed"
    exit 1
fi

# Test 5: Verify non-root user
echo
echo "Test 5: Verify non-root user..."
USER_ID=$(docker run --rm "$IMAGE" id -u)
if [ "$USER_ID" = "1000" ]; then
    echo "✓ Running as UID 1000"
else
    echo "✗ Not running as UID 1000 (got: $USER_ID)"
    exit 1
fi

# Test 6: Verify shell execution
echo
echo "Test 6: Verify shell execution..."
OUTPUT=$(docker run --rm "$IMAGE" sh -c "echo 'test passed'")
if [ "$OUTPUT" = "test passed" ]; then
    echo "✓ Shell execution works"
else
    echo "✗ Shell execution failed"
    exit 1
fi

echo
echo "======================================"
echo "All tests passed! ✓"
