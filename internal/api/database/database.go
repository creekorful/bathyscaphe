package database

import (
	"context"
	"encoding/json"
	"github.com/olivere/elastic/v7"
	"github.com/rs/zerolog/log"
	"time"
)

//go:generate mockgen -destination=../database_mock/database_mock.go -package=database_mock . Database

var resourcesIndex = "resources"

// ResourceIdx represent a resource as stored in elasticsearch
type ResourceIdx struct {
	URL         string            `json:"url"`
	Body        string            `json:"body"`
	Time        time.Time         `json:"time"`
	Title       string            `json:"title"`
	Meta        map[string]string `json:"meta"`
	Description string            `json:"description"`
	Headers     map[string]string `json:"headers"`
}

// ResSearchParams is the search params used
type ResSearchParams struct {
	URL        string
	Keyword    string
	StartDate  time.Time
	EndDate    time.Time
	WithBody   bool
	PageSize   int
	PageNumber int
	// TODO allow searching by meta
	// TODO allow searching by headers
}

// Database is the interface used to abstract communication
// with the persistence unit
type Database interface {
	SearchResources(params *ResSearchParams) ([]ResourceIdx, error)
	CountResources(params *ResSearchParams) (int64, error)
	AddResource(res ResourceIdx) error
}

type elasticSearchDB struct {
	client *elastic.Client
}

// NewElasticDB create a new Database based on ES instance
func NewElasticDB(uri string) (Database, error) {
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

	return &elasticSearchDB{
		client: ec,
	}, nil
}

func (e *elasticSearchDB) SearchResources(params *ResSearchParams) ([]ResourceIdx, error) {
	q := buildSearchQuery(params)
	from := (params.PageNumber - 1) * params.PageSize

	res, err := e.client.Search().
		Index(resourcesIndex).
		Query(q).
		From(from).
		Size(params.PageSize).
		Do(context.Background())
	if err != nil {
		return nil, err
	}

	var resources []ResourceIdx
	for _, hit := range res.Hits.Hits {
		var resource ResourceIdx
		if err := json.Unmarshal(hit.Source, &resource); err != nil {
			log.Warn().Str("err", err.Error()).Msg("Error while un-marshaling resource")
			continue
		}

		// Remove body if not wanted
		if !params.WithBody {
			resource.Body = ""
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

func (e *elasticSearchDB) CountResources(params *ResSearchParams) (int64, error) {
	q := buildSearchQuery(params)

	count, err := e.client.Count(resourcesIndex).Query(q).Do(context.Background())
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (e *elasticSearchDB) AddResource(res ResourceIdx) error {
	_, err := e.client.Index().
		Index(resourcesIndex).
		BodyJson(res).
		Do(context.Background())
	return err
}

func buildSearchQuery(params *ResSearchParams) elastic.Query {
	var queries []elastic.Query
	if params.URL != "" {
		log.Trace().Str("url", params.URL).Msg("SearchQuery: Setting url")
		queries = append(queries, elastic.NewMatchQuery("url", params.URL))
	}
	if params.Keyword != "" {
		log.Trace().Str("body", params.Keyword).Msg("SearchQuery: Setting body")
		queries = append(queries, elastic.NewMatchQuery("body", params.Keyword))
	}
	if !params.StartDate.IsZero() || !params.EndDate.IsZero() {
		timeQuery := elastic.NewRangeQuery("time")

		if !params.StartDate.IsZero() {
			log.Trace().
				Str("startDate", params.StartDate.Format(time.RFC3339)).
				Msg("SearchQuery: Setting startDate")
			timeQuery.Gte(params.StartDate.Format(time.RFC3339))
		}
		if !params.EndDate.IsZero() {
			log.Trace().
				Str("endDate", params.EndDate.Format(time.RFC3339)).
				Msg("SearchQuery: Setting endDate")
			timeQuery.Lte(params.EndDate.Format(time.RFC3339))
		}
		queries = append(queries, timeQuery)
	}

	// Handle specific case
	if len(queries) == 0 {
		return elastic.NewMatchAllQuery()
	}
	if len(queries) == 1 {
		return queries[0]
	}

	// otherwise AND combine them
	return elastic.NewBoolQuery().Must(queries...)
}

func setupElasticSearch(ctx context.Context, es *elastic.Client) error {
	// Setup index if doesn't exist
	exist, err := es.IndexExists(resourcesIndex).Do(ctx)
	if err != nil {
		return err
	}
	if !exist {
		log.Debug().Str("index", resourcesIndex).Msg("Creating missing index")
		if _, err := es.CreateIndex(resourcesIndex).Do(ctx); err != nil {
			return err
		}
	}

	return nil
}
