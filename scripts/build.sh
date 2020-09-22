#!/bin/bash

# set image tag if provided
tag="latest"
if [ "$1" ]; then
  tag="$1"
fi

# build docker images
for path in build/docker/Dockerfile.*; do
  name=$(echo "$path" | cut -d'.' -f2)
  docker build . -f "$path" -t "creekorful/$name:$tag"
done
