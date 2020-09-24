# Trandoshan dark web crawler

![CI](https://github.com/creekorful/trandoshan/workflows/CI/badge.svg)

This repository is a complete rewrite of the Trandoshan dark web crawler. Everything has been written inside a single
Git repository to ease maintenance.

## Why a rewrite?

The first version of Trandoshan [(available here)](https://github.com/trandoshan-io) is working great but
not really professional, the code start to be a mess, hard to manage since split in multiple repositories, etc.

I have therefore decided to create & maintain the project in this specific repository, 
where all process code will be available (as a Go module).

# How to start the crawler

Execute the ``/scripts/start.sh`` and wait for all containers to start.
You can start the crawler in detached mode by passing --detach to it.

## Note

Ensure you have at least 3GB of memory as the Elasticsearch stack docker will require 2GB.

# How to start the crawling process

Since the API is exposed on localhost:15005, one can use it to start the crawling process:

using trandoshanctl executable:

```sh
$ trandoshanctl schedule https://www.facebookcorewwwi.onion
```

or using the docker image:

```sh
$ docker run creekorful/trandoshanctl --api-uri <uri> schedule https://www.facebookcorewwwi.onion
```

(you'll need to specify the api uri if you use the docker container) 

this will schedule given URL for crawling.

# How to view results

## Using trandoshanctl

```sh
trandoshanctl search <term>
```

## Using kibana

You can use the Kibana dashboard available at http://localhost:15004.
You will need to create an index pattern named 'resources', and when it asks for the time field, choose 'time'.

# How to hack the crawler

If you've made a change to one of the crawler process and wish to use the updated version when
running ``./scripts/start.sh`` you just need to run the ``./scripts/build.sh`` script.