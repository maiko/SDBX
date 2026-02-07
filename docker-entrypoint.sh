#!/bin/sh
set -e

# If the Docker socket exists, ensure the sdbx user can access it
# by matching the socket's group ID inside the container.
DOCKER_SOCKET="/var/run/docker.sock"
if [ -S "$DOCKER_SOCKET" ]; then
    DOCKER_GID=$(stat -c '%g' "$DOCKER_SOCKET")
    # Check if a group with this GID already exists
    if ! getent group "$DOCKER_GID" > /dev/null 2>&1; then
        addgroup -g "$DOCKER_GID" dockersock
    fi
    DOCKER_GROUP=$(getent group "$DOCKER_GID" | cut -d: -f1)
    addgroup sdbx "$DOCKER_GROUP" 2>/dev/null || true
fi

# Drop privileges and run as non-root user
exec su-exec sdbx "$@"
