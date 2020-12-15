#!/bin/bash

# set image tag if provided
tag="latest"
if [ "$1" ]; then
  tag="$1"
fi

# push docker images
for path in build/docker/Dockerfile.*; do
  name=$(echo "$path" | cut -d'.' -f2)
  docker push "creekorful/$name:$tag"
done
