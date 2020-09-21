package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/creekorful/trandoshan/internal/util/logging"
	natsutil "github.com/creekorful/trandoshan/internal/util/nats"
	"github.com/creekorful/trandoshan/pkg/proto"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var (
	protocolRegex = regexp.MustCompile("https?://")
)

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

func execute(ctx *cli.Context) error {
	logging.ConfigureLogger(ctx)

	e := echo.New()
	e.HideBanner = true

	log.Info().Str("ver", ctx.App.Version).Msg("Starting tdsh-api")

	log.Debug().Str("uri", ctx.String("elasticsearch-uri")).Msg("Using Elasticsearch server")
	log.Debug().Str("uri", ctx.String("nats-uri")).Msg("Using NATS server")

	// Connect to the NATS server
	nc, err := nats.Connect(ctx.String("nats-uri"))
	if err != nil {
		log.Err(err).Str("uri", ctx.String("nats-uri")).Msg("Error while connecting to NATS server")
		return err
	}
	defer nc.Close()

	// Create Elasticsearch client
	es, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{ctx.String("elasticsearch-uri")}})
	if err != nil {
		log.Err(err).Msg("Error while creating ES client")
		return err
	}

	// Add endpoints
	e.GET("/v1/resources", searchResources(es))
	e.POST("/v1/resources", addResource(es))
	e.POST("/v1/urls", addURL(nc))

	log.Info().Msg("Successfully initialized tdsh-api. Waiting for requests")

	return e.Start(":8080")
}

func searchResources(es *elasticsearch.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		b64URL := c.QueryParam("url")
		b, err := base64.URLEncoding.DecodeString(b64URL)
		if err != nil {
			log.Err(err).Msg("Error while decoding URL")
			return c.NoContent(http.StatusInternalServerError)
		}

		var buf bytes.Buffer
		query := map[string]interface{}{
			"query": map[string]interface{}{
				"match": map[string]interface{}{
					"url": string(b),
				},
			},
		}
		if err := json.NewEncoder(&buf).Encode(query); err != nil {
			log.Err(err).Msg("Error encoding query")
			return c.NoContent(http.StatusInternalServerError)
		}

		// Perform the search request.
		res, err := es.Search(
			es.Search.WithContext(context.Background()),
			es.Search.WithIndex("resources"),
			es.Search.WithBody(&buf),
		)
		if err != nil || (res.IsError() && res.StatusCode != http.StatusNotFound) {
			evt := log.Err(err)
			if res != nil {
				evt.Int("status", res.StatusCode)
			}
			evt.Msg("Error getting response from ES")

			return c.NoContent(http.StatusInternalServerError)
		}

		// In case the collection does not already exist
		// ES will return 404 NOT FOUND
		if res.StatusCode == http.StatusNotFound {
			return c.JSON(http.StatusOK, []proto.ResourceDto{})
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
			log.Err(err).Msg("Error parsing the response body from ES")
			return c.NoContent(http.StatusInternalServerError)
		}

		var urls []proto.ResourceDto
		for _, rawHit := range resp["hits"].(map[string]interface{})["hits"].([]interface{}) {
			rawSrc := rawHit.(map[string]interface{})["_source"].(map[string]interface{})

			res := proto.ResourceDto{
				URL:   rawSrc["url"].(string),
				Title: rawSrc["title"].(string),
			}

			t, err := time.Parse(time.RFC3339, rawSrc["time"].(string))
			if err == nil {
				res.Time = t
			}

			urls = append(urls, res)
		}

		return c.JSON(http.StatusOK, urls)
	}
}

func addResource(es *elasticsearch.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		var resourceDto proto.ResourceDto
		if err := json.NewDecoder(c.Request().Body).Decode(&resourceDto); err != nil {
			log.Err(err).Msg("Error while un-marshaling resource")
			return c.NoContent(http.StatusUnprocessableEntity)
		}

		log.Debug().Str("url", resourceDto.URL).Msg("Saving resource")

		// TODO store on file system

		// Create Elasticsearch document
		doc := resourceIndex{
			URL:   protocolRegex.ReplaceAllLiteralString(resourceDto.URL, ""),
			Body:  resourceDto.Body,
			Title: extractTitle(resourceDto.Body),
			Time:  time.Now(),
		}

		// Serialize document into json
		docBytes, err := json.Marshal(&doc)
		if err != nil {
			log.Err(err).Msg("Error while serializing document into json")
			return c.NoContent(http.StatusInternalServerError)
		}

		// Use Elasticsearch to index document
		req := esapi.IndexRequest{
			Index:   "resources",
			Body:    bytes.NewReader(docBytes),
			Refresh: "true",
		}
		res, err := req.Do(context.Background(), es)
		if err != nil {
			log.Err(err).Msg("Error while creating elasticsearch index")
			return c.NoContent(http.StatusInternalServerError)
		}
		defer res.Body.Close()

		log.Debug().Str("url", resourceDto.URL).Msg("Successfully saved resource")

		return c.NoContent(http.StatusCreated)
	}
}

func addURL(nc *nats.Conn) echo.HandlerFunc {
	return func(c echo.Context) error {
		var url string
		if err := json.NewDecoder(c.Request().Body).Decode(&url); err != nil {
			log.Err(err).Msg("Error while un-marshaling URL")
			return c.NoContent(http.StatusUnprocessableEntity)
		}

		// Publish the URL
		if err := natsutil.PublishJSON(nc, proto.URLFoundSubject, &proto.URLFoundMsg{URL: url}); err != nil {
			log.Err(err).Msg("Unable to publish URL")
			return c.NoContent(http.StatusInternalServerError)
		}

		log.Debug().Str("url", url).Msg("Successfully published URL")

		return nil
	}
}

// extract title from html body
func extractTitle(body string) string {
	cleanBody := strings.ToLower(body)

	if strings.Index(cleanBody, "<title>") == -1 || strings.Index(cleanBody, "</title>") == -1 {
		return ""
	}

	// TODO improve
	startPos := strings.Index(cleanBody, "<title>") + len("<title>")
	endPos := strings.Index(cleanBody, "</title>")

	return body[startPos:endPos]
}
