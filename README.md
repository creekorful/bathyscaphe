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
## Note

Ensure you have at least 3GB of memory as the Elasticsearch stack docker will require 2GB.

## Prepare the feeder process

The feeder process is used to send the first URL to the crawler.  By default it's commented out in: /deployments/docker-compose.yml 
 
 ```#feeder:
  #  image: trandoshan.io/feeder:latest
  #  command: --log-level debug --nats-uri nats --url https://www.facebookcorewwwi.onion
 ```
  
Un-comment it, and set an appropriate URL.

## Start the crawler

Execute the ``/scripts/start.sh`` and wait for all containers to start.
You should see the crawling process in action.

## Note

Sometimes the crawling process may not start, this is because the feeder process is starting before the others are online. In that case wait for all processes to be online, then execute the following command:

```
docker-compose run feeder
```

this will manually restart the feeder.

# Access the Kibana UI to view results

Now head out to http://localhost:15004

You will need to create an index pattern named 'resources', and when it asks for the time field, choose 'time'.
