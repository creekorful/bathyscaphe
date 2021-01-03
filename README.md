# Trandoshan dark web crawler

![CI](https://github.com/creekorful/trandoshan/workflows/CI/badge.svg)

This repository is a complete rewrite of the Trandoshan dark web crawler. Everything has been written inside a single
Git repository to ease maintenance.

## Why a rewrite?

The first version of Trandoshan [(available here)](https://github.com/trandoshan-io) is working great but not really
professional, the code start to be a mess, hard to manage since split in multiple repositories, etc.

I have therefore decided to create & maintain the project in this specific repository, where all components code will be
available (as a Go module).

# How to start the crawler

To start the crawler, one just need to execute the following command:

```sh
$ ./scripts/start.sh
```

and wait for all containers to start.

## Notes

- You can start the crawler in detached mode by passing --detach to start.sh.
- Ensure you have at least 3 GB of memory as the Elasticsearch stack docker will require 2 GB.

# How to initiate crawling

Since the API is exposed on localhost:15005, one can use it to start crawling:

using trandoshanctl executable:

```sh
$ trandoshanctl --api-uri <uri> schedule https://www.facebookcorewwwi.onion
```

or using the docker image:

```sh
$ docker run creekorful/trandoshanctl --api-uri <uri> schedule https://www.facebookcorewwwi.onion
```

(you'll need to specify the api uri if you use the docker container)

this will schedule given URL for crawling.

## How to speed up crawling

If one want to speed up the crawling, he can scale the instance of crawling component in order to increase performances.
This may be done by issuing the following command after the crawler is started:

```sh
$ ./scripts/scale.sh crawler=5
```

this will set the number of crawler instance to 5.

# How to view results

## Using trandoshanctl

```sh
$ trandoshanctl search <term>
```

## Using kibana

You can use the Kibana dashboard available at http://localhost:15004. You will need to create an index pattern named '
resources', and when it asks for the time field, choose 'time'.

# How to hack the crawler

If you've made a change to one of the crawler component and wish to use the updated version when running start.sh you
just need to issue the following command:

```sh
$ ./script/build.sh
```

this will rebuild all crawler images using local changes. After that just run start.sh again to have the updated version
running.

# Architecture

The architecture details are available [here](docs/architecture.png).

