#!/bin/bash

set -e

default_repo_name="zfslocal"
default_repo_path="/home/markus/.custom/zfs"
REPOSITORY_NAME=${REPOSITORY_NAME:-$default_repo_name}
REPOSITORY_PATH=${REPOSITORY_PATH:-$default_repo_path}


for file in $REPOSITORY_PATH/*.pkg.*; do
    repo-add "$REPOSITORY_PATH/$REPOSITORY_NAME.db.tar.zst" "$file" 
done