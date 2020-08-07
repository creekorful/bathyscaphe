#!/bin/bash

# build processes
for path in build/Dockerfile-*; do
  name=$(echo "$path" | cut -d'-' -f2)
  docker build . -f "$path" -t "trandoshan.io/$name"
done
