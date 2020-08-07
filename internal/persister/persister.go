package persister

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/creekorful/trandoshan/internal/api"
	"github.com/creekorful/trandoshan/internal/log"
	"github.com/creekorful/trandoshan/internal/natsutil"
	"github.com/creekorful/trandoshan/pkg/proto"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"net/http"
)

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
	logrus.Debugf("Using API server at: %s", ctx.String("api-uri"))

	// Create the HTTP client
	httpClient := &http.Client{}

	// Create the NATS subscriber
	sub, err := natsutil.NewSubscriber(ctx.String("nats-uri"))
	if err != nil {
		return err
	}
	defer sub.Close()

	logrus.Info("Successfully initialized trandoshan-persister. Waiting for resources")

	if err := sub.QueueSubscribe(proto.ResourceSubject, "persisters", handleMessage(httpClient, ctx.String("api-uri"))); err != nil {
		return err
	}

	return nil
}

func handleMessage(httpClient *http.Client, apiURI string) natsutil.MsgHandler {
	return func(nc *nats.Conn, msg *nats.Msg) error {
		var resMsg proto.ResourceMsg
		if err := natsutil.ReadJSON(msg, &resMsg); err != nil {
			return err
		}

		logrus.Debugf("Processing resource: %s", resMsg.URL)

		body, err := json.Marshal(&api.ResourceDto{
			URL:  resMsg.URL,
			Body: resMsg.Body,
		})

		if err != nil {
			logrus.Errorf("Error while serializing resource: %s", err)
			return err
		}

		url := fmt.Sprintf("%s/v1/resources", apiURI)
		logrus.Tracef("Posting on API URL: %s", url)

		resp, err := httpClient.Post(url, "application/json", bytes.NewBuffer(body))
		if err != nil || resp.StatusCode != http.StatusCreated {
			logrus.Errorf("Error while sending resource to the API: %s", err)
			logrus.Errorf("Received status code: %d", resp.StatusCode)
			return err
		}

		logrus.Debugf("Successfully processed resource: %s", resMsg.URL)

		return nil
	}
}
