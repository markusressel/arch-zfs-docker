#!/bin/bash

default_repo_name="zfslocal"
default_repo_path="/home/markus/.custom/zfs"
REPOSITORY_NAME=${REPOSITORY_NAME:-$default_repo_name}
REPOSITORY_PATH=${REPOSITORY_PATH:-$default_repo_path}

default_variant=""
VARIANT=${VARIANT:-$default_variant}

REPOSITORY_PATH=$REPOSITORY_PATH VARIANT=$VARIANT ./run-container.sh

REPOSITORY_NAME=$REPOSITORY_NAME REPOSITORY_PATH=$REPOSITORY_PATH VARIANT=$VARIANT ./update-local-package-repository.sh
