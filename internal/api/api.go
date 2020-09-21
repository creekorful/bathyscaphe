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
	es, err := elastic.NewClient(
		elastic.SetURL(ctx.String("elasticsearch-uri")),
		elastic.SetHealthcheck(false),
	)
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

func searchResources(es *elastic.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		// First of all base64decode the URL
		b64URL := c.QueryParam("url")
		b, err := base64.URLEncoding.DecodeString(b64URL)
		if err != nil {
			log.Err(err).Msg("Error while decoding URL")
			return c.NoContent(http.StatusInternalServerError)
		}

		// Perform the search request.
		query := elastic.NewMatchQuery("url", string(b))
		res, err := es.Search().
			Index("resource").
			Query(query).
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
			resources = append(resources, resource)
		}

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
			URL:   protocolRegex.ReplaceAllLiteralString(resourceDto.URL, ""),
			Body:  resourceDto.Body,
			Title: extractTitle(resourceDto.Body),
			Time:  time.Now(),
		}

		_, err := es.Index().
			Index("resources").
			BodyJson(doc).
			Do(context.Background())
		if err != nil {
			log.Err(err).Msg("Error while creating ES document")
			return err
		}

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
		if err := natsutil.PublishMsg(nc, &messaging.URLFoundMsg{URL: url}); err != nil {
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
