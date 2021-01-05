package index

import (
	"context"
	"github.com/PuerkitoBio/goquery"
	"github.com/olivere/elastic/v7"
	"github.com/rs/zerolog/log"
	"strings"
	"time"
)

var resourcesIndex = "resources"

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

func (e *elasticSearchIndex) IndexResource(url string, time time.Time, body string, headers map[string]string) error {
	res, err := extractResource(url, time, body, headers)
	if err != nil {
		return err
	}

	_, err = e.client.Index().
		Index(resourcesIndex).
		BodyJson(res).
		Do(context.Background())
	return err
}

func setupElasticSearch(ctx context.Context, es *elastic.Client) error {
	// Setup index if doesn't exist
	exist, err := es.IndexExists(resourcesIndex).Do(ctx)
	if err != nil {
		return err
	}
	if !exist {
		log.Debug().Str("index", resourcesIndex).Msg("Creating missing index")

		q := es.CreateIndex(resourcesIndex).BodyString(mapping)
		if _, err := q.Do(ctx); err != nil {
			return err
		}
	}

	return nil
}

func extractResource(url string, time time.Time, body string, headers map[string]string) (*resourceIdx, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
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
	for key, value := range headers {
		lowerCasedHeaders[strings.ToLower(key)] = value
	}

	return &resourceIdx{
		URL:         url,
		Body:        body,
		Time:        time,
		Title:       title,
		Meta:        meta,
		Description: meta["description"],
		Headers:     lowerCasedHeaders,
	}, nil
}
