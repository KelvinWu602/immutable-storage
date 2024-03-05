#!/bin/bash

# This shell script will start an IPFS daemon on localhost for testing
# Run this script in WSL Ubuntu if you are on Windows

MOUNT_DATA="$HOME/immutable-storage-local-ipfs-mounts/data"
MOUNT_STAGING="$HOME/immutable-storage-local-ipfs-mounts/staging"

# Create stateful folders if not exists
mkdir -p $MOUNT_DATA
mkdir -p $MOUNT_STAGING

# Start daemon
docker run -d --name ipfs-daemon -v $MOUNT_STAGING:/export -v $MOUNT_STAGING:/data/ipfs -p 4001:4001 -p 4001:4001/udp -p 127.0.0.1:8080:8080 -p 127.0.0.1:5001:5001 ipfs/kubo:latest