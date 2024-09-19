#!/bin/bash

default_repo_path="/home/markus/.custom/zfs"
REPOSITORY_PATH=${REPOSITORY_PATH:-$default_repo_path}

default_variant=""
VARIANT=${VARIANT:-$default_variant}

mkdir -p "$REPOSITORY_PATH"

sudo docker buildx build --tag archbuild --build-arg VARIANT="$VARIANT" .
mkdir -p "$REPOSITORY_PATH"
sudo docker run -i -t --rm -v $REPOSITORY_PATH:/packages archbuild


