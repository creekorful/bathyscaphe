#!/bin/bash

# build docker images
for path in build/docker/Dockerfile-*; do
  name=$(echo "$path" | cut -d'-' -f2)
  docker build . -f "$path" -t "creekorful/tdsh-$name:latest"
done
