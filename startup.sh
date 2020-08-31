#!/bin/sh

if [ "$MONGODB_URL" == "" ]; then
    echo "MONGODB_URL is required"
    exit 1
fi

mongos-exporter --mongodb-url=$MONGODB_URL

