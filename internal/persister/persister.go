package persister

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/creekorful/trandoshan/internal/log"
	"github.com/creekorful/trandoshan/internal/natsutil"
	"github.com/creekorful/trandoshan/pkg/proto"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"strings"
	"time"
)

type resourceIndex struct {
	URL   string    `json:"url"`
	Body  string    `json:"body"`
	Title string    `json:"title"`
	Time  time.Time `json:"time"`
}

// GetApp return the persister app
func GetApp() *cli.App {
	return &cli.App{
		Name:    "trandoshan-persister",
		Version: "0.0.1",
		Usage:   "", // TODO
		Flags: []cli.Flag{
			log.GetLogFlag(),
			&cli.StringFlag{
				Name:     "nats-uri",
				Usage:    "URI to the NATS server",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "api-uri",
				Usage:    "URI to the API server",
				Required: true,
			},
		},
		Action: execute,
	}
}

func execute(ctx *cli.Context) error {
	log.ConfigureLogger(ctx)

	logrus.Infof("Starting trandoshan-persister v%s", ctx.App.Version)

	logrus.Debugf("Using NATS server at: %s", ctx.String("nats-uri"))

	// Create Elasticsearch client
	es, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{ctx.String("elasticsearch-uri")}})
	if err != nil {
		logrus.Errorf("Error while creating elasticsearch client: %s", err)
		return err
	}

	// Create the NATS subscriber
	sub, err := natsutil.NewSubscriber(ctx.String("nats-uri"))
	if err != nil {
		return err
	}
	defer sub.Close()

	logrus.Info("Successfully initialized trandoshan-persister. Waiting for resources")

	if err := sub.QueueSubscribe(proto.ResourceSubject, "persisters", handleMessage(es)); err != nil {
		return err
	}

	return nil
}

func handleMessage(es *elasticsearch.Client) natsutil.MsgHandler {
	return func(nc *nats.Conn, msg *nats.Msg) error {
		var resMsg proto.ResourceMsg
		if err := natsutil.ReadJSON(msg, &resMsg); err != nil {
			return err
		}

		logrus.Debugf("Processing resource: %s", resMsg.URL)

		// TODO store on file system

		// Create Elasticsearch document
		doc := resourceIndex{
			URL:   resMsg.URL,
			Body:  resMsg.Body,
			Title: extractTitle(resMsg.Body),
			Time:  time.Now(),
		}

		// Serialize document into json
		docBytes, err := json.Marshal(&doc)
		if err != nil {
			logrus.Warnf("Error while serializing document into json: %s", err)
		}

		// Use Elasticsearch to index document
		req := esapi.IndexRequest{
			Index:   "resources",
			Body:    bytes.NewReader(docBytes),
			Refresh: "true",
		}
		res, err := req.Do(context.Background(), es)
		if err != nil {
			logrus.Warnf("Error while creating elasticsearch index: %s", err)
		}
		defer res.Body.Close()

		return nil
	}
}

// extract title from html body
func extractTitle(body string) string {
	cleanBody := strings.ToLower(body)
	startPos := strings.Index(cleanBody, "<title>") + len("<title>")
	endPos := strings.Index(cleanBody, "</title>")

	// html tag absent of malformed
	if startPos == -1 || endPos == -1 {
		return ""
	}
	return body[startPos:endPos]
}
