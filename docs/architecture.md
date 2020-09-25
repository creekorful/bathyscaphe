# Crawler

The crawler is the central component of Trandoshan.
It consumes URL, crawl them and publish the page body while following redirects etc...

## Consumes

- URL (url.todo)

## Produces

- Resource (resource.new)

# Extractor

The extractor is the data extraction component of Trandoshan.
It consumes crawled resource, extract data (urls, metadata, etc...) from it,
store them into an ES instance (by calling the API), & publish found URLs.

## Consumes

- Resource (resource.new)

## Produces

- URL (url.found)
- Metadata
- Body

# Scheduler

The scheduler is the component responsible for crawling schedule part.
It determinates which URL should be crawled and publish them.

## Consumes

- URL (url.found)

## Produces

- URL (url.todo)

# API

The API component is mainly used to get data from ES.