# Trandoshan dark web crawler

[![trandoshan](https://snapcraft.io//trandoshan/badge.svg)](https://snapcraft.io/trandoshan)

This repository is a complete rewrite of the Trandoshan dark web crawler. Everything has been written inside a single
Git repository to ease maintenance.

## Why a rewrite?

The first version of Trandoshan [(available here)](https://github.com/trandoshan-io) is working great but
not really professional, the code start to be a mess, hard to manage since split in multiple repositories, etc..

I have therefore decided to create & maintain the project in this specific directory, where all process code will be available
(as a Go module).

# How build the crawler

Since the docker image are not available yet, one must run the following script in order to build the crawler fully.

```sh
./scripts/build.sh
```

# How to start the crawler

Execute the ``/scripts/start.sh`` and wait for all containers to start.
You can start the crawler in detached mode by passing --detach to start.sh

## Note

Ensure you have at least 3GB of memory as the Elasticsearch stack docker will require 2GB.

# How to start the crawling process

Since the API is explosed on localhost:15005, one can use it to start the crawling process:

```sh
feeder --api-uri http://localhost:15005 --url https://www.facebookcorewwwi.onion
```

this will 'force' the API to publish given URL in crawling queue.

## How to view results

At the moment there is no Trandoshan dashboard.
You can use the Kibana dashboard available at http://localhost:15004.

You will need to create an index pattern named 'resources', and when it asks for the time field, choose 'time'.
