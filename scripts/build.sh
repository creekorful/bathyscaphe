#!/bin/bash

docker build . -f build/Dockerfile-crawler -t trandoshan.io/crawler
docker build . -f build/Dockerfile-feeder -t trandoshan.io/feeder
