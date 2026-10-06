#!/bin/bash

default_repo_path="/home/markus/.custom/zfs"
REPOSITORY_PATH=${REPOSITORY_PATH:-$default_repo_path}

default_variant=""
VARIANT=${VARIANT:-$default_variant}

mkdir -p "$REPOSITORY_PATH"

sudo docker buildx build -f Dockerfile-archzfs --tag archzfs .
mkdir -p "$REPOSITORY_PATH"

if [[ -v GITHUB_ACTIONS ]]; then
  sudo docker run --rm -v "$REPOSITORY_PATH:/packages" archzfs
else
  sudo docker run -i -t --rm -v "$REPOSITORY_PATH:/packages" archzfs
fi
