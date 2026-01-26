# Minimal Alpine-based image for overlay network testing
# Provides: bash, ping, and minimal runtime (~10MB compressed)
FROM alpine:latest

# Install only required packages
# - bash: for shell execution
# - iputils: for ping command
RUN apk add --no-cache \
    bash \
    iputils && \
    rm -rf /var/cache/apk/*

# Create non-root user and group
RUN addgroup -g 1000 overlaytest && \
    adduser -D -u 1000 -G overlaytest -s /bin/bash overlaytest

# Switch to non-root user
USER 1000:1000

# Keep container running (will be overridden by DaemonSet)
CMD ["/bin/bash", "-c", "tail -f /dev/null"]

# Labels for metadata
LABEL org.opencontainers.image.title="overlaytest-image"
LABEL org.opencontainers.image.description="Minimal container image for Kubernetes overlay network testing"
LABEL org.opencontainers.image.source="https://github.com/eumel8/overlaytest"
#LABEL org.opencontainers.image.licenses="Apache-2.0"
