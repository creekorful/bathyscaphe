package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/creekorful/trandoshan/api"
	"github.com/creekorful/trandoshan/internal/messaging"
	"github.com/creekorful/trandoshan/internal/util/logging"
	natsutil "github.com/creekorful/trandoshan/internal/util/nats"
	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"
	"github.com/olivere/elastic/v7"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"net/http"
	"strconv"
	"time"
)

var (
	resourcesIndex = "resources"

	paginationPageHeader     = "X-Pagination-Page"
	paginationSizeHeader     = "X-Pagination-Size"
	paginationCountHeader    = "X-Pagination-Count"
	paginationPageQueryParam = "pagination-page"
	paginationSizeQueryParam = "pagination-size"

	defaultPaginationSize = 50
	maxPaginationSize     = 100
)

type pagination struct {
	page int
	size int
}

// Represent a resource in elasticsearch
type resourceIndex struct {
	URL   string    `json:"url"`
	Body  string    `json:"body"`
	Title string    `json:"title"`
	Time  time.Time `json:"time"`
}

// GetApp return the api app
func GetApp() *cli.App {
	return &cli.App{
		Name:    "tdsh-api",
		Version: "0.3.0",
		Usage:   "Trandoshan API process",
		Flags: []cli.Flag{
			logging.GetLogFlag(),
			&cli.StringFlag{
				Name:     "nats-uri",
				Usage:    "URI to the NATS server",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "elasticsearch-uri",
				Usage:    "URI to the Elasticsearch server",
				Required: true,
			},
		},
		Action: execute,
	}
}

func execute(c *cli.Context) error {
	logging.ConfigureLogger(c)

	e := echo.New()
	e.HideBanner = true

	log.Info().Str("ver", c.App.Version).Msg("Starting tdsh-api")

	log.Debug().Str("uri", c.String("elasticsearch-uri")).Msg("Using Elasticsearch server")
	log.Debug().Str("uri", c.String("nats-uri")).Msg("Using NATS server")

	// Connect to the NATS server
	nc, err := nats.Connect(c.String("nats-uri"))
	if err != nil {
		log.Err(err).Str("uri", c.String("nats-uri")).Msg("Error while connecting to NATS server")
		return err
	}
	defer nc.Close()

	// Create Elasticsearch client
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	es, err := elastic.DialContext(ctx,
		elastic.SetURL(c.String("elasticsearch-uri")),
		elastic.SetSniff(false),
		elastic.SetHealthcheck(false),
	)
	if err != nil {
		log.Err(err).Msg("Error while creating ES client")
		return err
	}

	// Setup ES one for all
	if err := setupElasticSearch(ctx, es); err != nil {
		return err
	}

	// Add endpoints
	e.GET("/v1/resources", searchResources(es))
	e.POST("/v1/resources", addResource(es))
	e.POST("/v1/urls", scheduleURL(nc))

	log.Info().Msg("Successfully initialized tdsh-api. Waiting for requests")

	return e.Start(":8080")
}

func searchResources(es *elastic.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		withBody := false
		if c.QueryParam("with-body") == "true" {
			withBody = true
		}

		startDate := time.Time{}
		if val := c.QueryParam("start-date"); val != "" {
			d, err := time.Parse(time.RFC3339, val)
			if err == nil {
				startDate = d
			}
		}

		endDate := time.Time{}
		if val := c.QueryParam("end-date"); val != "" {
			d, err := time.Parse(time.RFC3339, val)
			if err == nil {
				endDate = d
			}
		}

		// First of all base64decode the URL
		b64URL := c.QueryParam("url")
		b, err := base64.URLEncoding.DecodeString(b64URL)
		if err != nil {
			log.Err(err).Msg("Error while decoding URL")
			return c.NoContent(http.StatusUnprocessableEntity)
		}

		// Acquire pagination
		p := readPagination(c)
		from := (p.page - 1) * p.size

		// Build up search query
		query := buildSearchQuery(string(b), c.QueryParam("keyword"), startDate, endDate)

		// Get total count
		totalCount, err := es.Count(resourcesIndex).Query(query).Do(context.Background())
		if err != nil {
			log.Err(err).Msg("Error while counting on ES")
			return c.NoContent(http.StatusInternalServerError)
		}

		// Perform the search request.
		res, err := es.Search().
			Index(resourcesIndex).
			Query(query).
			From(from).
			Size(p.size).
			Do(context.Background())
		if err != nil {
			log.Err(err).Msg("Error while searching on ES")
			return c.NoContent(http.StatusInternalServerError)
		}

		var resources []api.ResourceDto
		for _, hit := range res.Hits.Hits {
			var resource api.ResourceDto
			if err := json.Unmarshal(hit.Source, &resource); err != nil {
				log.Warn().Str("err", err.Error()).Msg("Error while un-marshaling resource")
				continue
			}

			// Remove body if not wanted
			if !withBody {
				resource.Body = ""
			}

			resources = append(resources, resource)
		}

		// Write pagination
		writePagination(c, p, totalCount)

		return c.JSON(http.StatusOK, resources)
	}
}

func addResource(es *elastic.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		var resourceDto api.ResourceDto
		if err := json.NewDecoder(c.Request().Body).Decode(&resourceDto); err != nil {
			log.Err(err).Msg("Error while un-marshaling resource")
			return c.NoContent(http.StatusUnprocessableEntity)
		}

		log.Debug().Str("url", resourceDto.URL).Msg("Saving resource")

		// Create Elasticsearch document
		doc := resourceIndex{
			URL:   resourceDto.URL,
			Body:  resourceDto.Body,
			Title: resourceDto.Title,
			Time:  resourceDto.Time,
		}

		_, err := es.Index().
			Index(resourcesIndex).
			BodyJson(doc).
			Do(context.Background())
		if err != nil {
			log.Err(err).Msg("Error while creating ES document")
			return err
		}

		log.Debug().Str("url", resourceDto.URL).Msg("Successfully saved resource")

		return c.JSON(http.StatusCreated, resourceDto)
	}
}

func buildSearchQuery(url, keyword string, startDate, endDate time.Time) elastic.Query {
	var queries []elastic.Query
	if url != "" {
		log.Trace().Str("url", url).Msg("SearchQuery: Setting url")
		queries = append(queries, elastic.NewTermQuery("url", url))
	}
	if keyword != "" {
		log.Trace().Str("body", keyword).Msg("SearchQuery: Setting body")
		queries = append(queries, elastic.NewTermQuery("body", keyword))
	}
	if !startDate.IsZero() || !endDate.IsZero() {
		timeQuery := elastic.NewRangeQuery("time")

		if !startDate.IsZero() {
			log.Trace().Str("startDate", startDate.Format(time.RFC3339)).Msg("SearchQuery: Setting startDate")
			timeQuery.Gte(startDate.Format(time.RFC3339))
		}
		if !endDate.IsZero() {
			log.Trace().Str("endDate", endDate.Format(time.RFC3339)).Msg("SearchQuery: Setting endDate")
			timeQuery.Lte(endDate.Format(time.RFC3339))
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

func scheduleURL(nc *nats.Conn) echo.HandlerFunc {
	return func(c echo.Context) error {
		var url string
		if err := json.NewDecoder(c.Request().Body).Decode(&url); err != nil {
			log.Err(err).Msg("Error while un-marshaling URL")
			return c.NoContent(http.StatusUnprocessableEntity)
		}

		// Publish the URL
		if err := natsutil.PublishMsg(nc, &messaging.URLFoundMsg{URL: url}); err != nil {
			log.Err(err).Msg("Unable to publish URL")
			return c.NoContent(http.StatusInternalServerError)
		}

		log.Debug().Str("url", url).Msg("Successfully published URL")

		return nil
	}
}
func setupElasticSearch(ctx context.Context, es *elastic.Client) error {
	// Setup index if doesn't exist
	exist, err := es.IndexExists(resourcesIndex).Do(ctx)
	if err != nil {
		log.Err(err).Str("index", resourcesIndex).Msg("Error while checking if index exist")
		return err
	}
	if !exist {
		log.Debug().Str("index", resourcesIndex).Msg("Creating missing index")
		if _, err := es.CreateIndex(resourcesIndex).Do(ctx); err != nil {
			log.Err(err).Str("index", resourcesIndex).Msg("Error while creating index")
			return err
		}
	} else {
		log.Debug().Msg("index exist")
	}

	return nil
}

func readPagination(c echo.Context) pagination {
	paginationPage, err := strconv.Atoi(c.QueryParam(paginationPageQueryParam))
	if err != nil {
		paginationPage = 1
	}
	paginationSize, err := strconv.Atoi(c.QueryParam(paginationSizeQueryParam))
	if err != nil {
		paginationSize = defaultPaginationSize
	}
	// Prevent too much results from being returned
	if paginationSize > maxPaginationSize {
		paginationSize = maxPaginationSize
	}

	return pagination{
		page: paginationPage,
		size: paginationSize,
	}
}

func writePagination(c echo.Context, p pagination, totalCount int64) {
	c.Response().Header().Set(paginationPageHeader, strconv.Itoa(p.page))
	c.Response().Header().Set(paginationSizeHeader, strconv.Itoa(p.size))
	c.Response().Header().Set(paginationCountHeader, strconv.FormatInt(totalCount, 10))
}
