# Trandoshan dark web crawler

This repository is a complete rewrite of the Trandoshan dark web crawler. Everything has been written inside a single
Git repository to ease maintenance.

## Why a rewrite?

The first version of Trandoshan [(available here)](https://github.com/trandoshan-io) is working great but
not really professional, the code start to be a mess, hard to manage since split in multiple repositories, etc..

I have therefore decided to create & maintain the project in this specific directory, where all process code will be available
(as a Go module).

## How to start the crawler

Since the docker image are not available yet, one must run the following script in order to build the crawler fully.

```shell script
./scripts/build.sh
```

The crawler can be started using the start script:

```shell script
./scripts/start.sh
```