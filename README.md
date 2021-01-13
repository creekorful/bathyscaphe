# Bathyscaphe dark web crawler

![CI](https://github.com/darkspot-org/bathyscaphe/workflows/CI/badge.svg)

Bathyscaphe is a Go written, fast, highly configurable, cloud-native dark web crawler.

# How to start the crawler

To start the crawler, one just need to execute the following command:

```sh
$ ./scripts/docker/start.sh
```

and wait for all containers to start.

## Notes

- You can start the crawler in detached mode by passing --detach to start.sh.
- Ensure you have at least 3 GB of memory as the Elasticsearch stack docker will require 2 GB.

# How to initiate crawling

One can use the RabbitMQ dashboard available at localhost:15003, and publish a new JSON object in the **crawlingQueue**
.

The object should look like this:

```json
{
  "url": "https://facebookcorewwwi.onion"
}
```

## How to speed up crawling

If one want to speed up the crawling, he can scale the instance of crawling component in order to increase performances.
This may be done by issuing the following command after the crawler is started:

```sh
$ ./scripts/docker/start.sh -d --scale crawler=5
```

this will set the number of crawler instance to 5.

# How to view results

You can use the Kibana dashboard available at http://localhost:15004. You will need to create an index pattern named '
resources', and when it asks for the time field, choose 'time'.

# How to hack the crawler

If you've made a change to one of the crawler component and wish to use the updated version when running start.sh you
just need to issue the following command:

```sh
$ goreleaser --snapshot --skip-publish --rm-dist
```

this will rebuild all images using local changes. After that just run start.sh again to have the updated version
running.

# Architecture

The architecture details are available [here](docs/architecture.png).

