#!/bin/bash

MOUNT_DATA="~/immutable-storage-local-ipfs-mounts/data"
MOUNT_STAGING="~/immutable-storage-local-ipfs-mounts/staging"

# remove stateful folders
if [ -d "$MOUNT_DATA" ]; then
    rm -rf $MOUNT_DATA
else
    echo "$MOUNT_DATA does not exist"
fi

if [ -d "$MOUNT_STAGING" ]; then
    rm -rf $MOUNT_STAGING
else
    echo "$MOUNT_STAGING does not exist"
fi

