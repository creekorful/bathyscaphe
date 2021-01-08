package index

import (
	"context"
	"github.com/PuerkitoBio/goquery"
	"github.com/olivere/elastic/v7"
	"github.com/rs/zerolog/log"
	"strings"
	"time"
)

const resourcesIndexName = "resources"
const mapping = `
{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0
  },
  "mappings": {
    "dynamic": false,
    "properties": {
      "body": {
        "type": "text"
      },
      "description": {
        "type": "text"
      },
      "url": {
        "type": "text",
        "fields": {
          "keyword": {
            "type": "keyword"
          }
        }
      },
      "time": {
        "type": "date"
      },
      "title": {
        "type": "text"
      },
      "headers": {
        "properties": {
          "server": {
            "type": "text",
            "fields": {
              "keyword": {
                "type": "keyword"
              }
            }
          }
        }
      }
    }
  }
}`

type resourceIdx struct {
	URL         string            `json:"url"`
	Body        string            `json:"body"`
	Time        time.Time         `json:"time"`
	Title       string            `json:"title"`
	Meta        map[string]string `json:"meta"`
	Description string            `json:"description"`
	Headers     map[string]string `json:"headers"`
}

type elasticSearchIndex struct {
	client *elastic.Client
}

func newElasticIndex(uri string) (Index, error) {
	// Create Elasticsearch client
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ec, err := elastic.DialContext(ctx,
		elastic.SetURL(uri),
		elastic.SetSniff(false),
		elastic.SetHealthcheck(false),
	)
	if err != nil {
		return nil, err
	}

	if err := setupElasticSearch(ctx, ec); err != nil {
		return nil, err
	}

	return &elasticSearchIndex{
		client: ec,
	}, nil
}

func (e *elasticSearchIndex) IndexResource(resource Resource) error {
	res, err := indexResource(resource)
	if err != nil {
		return err
	}

	_, err = e.client.Index().
		Index(resourcesIndexName).
		BodyJson(res).
		Do(context.Background())
	return err
}

func (e *elasticSearchIndex) IndexResources(resources []Resource) error {
	bulkRequest := e.client.Bulk()

	for _, resource := range resources {
		resourceIndex, err := indexResource(resource)
		if err != nil {
			return err
		}

		req := elastic.NewBulkIndexRequest().
			Index(resourcesIndexName).
			Doc(resourceIndex)
		bulkRequest.Add(req)
	}

	_, err := bulkRequest.Do(context.Background())
	return err
}

func setupElasticSearch(ctx context.Context, es *elastic.Client) error {
	// Setup index if doesn't exist
	exist, err := es.IndexExists(resourcesIndexName).Do(ctx)
	if err != nil {
		return err
	}
	if !exist {
		log.Debug().Str("index", resourcesIndexName).Msg("Creating missing index")

		q := es.CreateIndex(resourcesIndexName).BodyString(mapping)
		if _, err := q.Do(ctx); err != nil {
			return err
		}
	}

	return nil
}

func indexResource(resource Resource) (*resourceIdx, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(resource.Body))
	if err != nil {
		return nil, err
	}

	// Get resource title
	title := doc.Find("title").First().Text()

	// Get meta values
	meta := map[string]string{}
	doc.Find("meta").Each(func(i int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		value, _ := s.Attr("content")

		// if name is empty then try to lookup using property
		if name == "" {
			name, _ = s.Attr("property")
			if name == "" {
				return
			}
		}

		meta[strings.ToLower(name)] = value
	})

	// Lowercase headers
	lowerCasedHeaders := map[string]string{}
	for key, value := range resource.Headers {
		lowerCasedHeaders[strings.ToLower(key)] = value
	}

	return &resourceIdx{
		URL:         resource.URL,
		Body:        resource.Body,
		Time:        resource.Time,
		Title:       title,
		Meta:        meta,
		Description: meta["description"],
		Headers:     lowerCasedHeaders,
	}, nil
}
