package scheduler

import (
	"encoding/base64"
	"fmt"
	"github.com/PuerkitoBio/purell"
	"github.com/creekorful/trandoshan/internal/util/http"
	"github.com/creekorful/trandoshan/internal/util/log"
	natsutil "github.com/creekorful/trandoshan/internal/util/nats"
	"github.com/creekorful/trandoshan/pkg/proto"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"net/url"
	"strings"
)

// GetApp return the scheduler app
func GetApp() *cli.App {
	return &cli.App{
		Name:    "tdsh-scheduler",
		Version: "0.2.0",
		Usage:   "Trandoshan scheduler process",
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

	logrus.Infof("Starting tdsh-scheduler v%s", ctx.App.Version)

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

	logrus.Info("Successfully initialized tdsh-scheduler. Waiting for URLs")

	if err := sub.QueueSubscribe(proto.URLFoundSubject, "schedulers", handleMessage(httpClient, ctx.String("api-uri"))); err != nil {
		return err
	}

	return nil
}

func handleMessage(httpClient *http.Client, apiURI string) natsutil.MsgHandler {
	return func(nc *nats.Conn, msg *nats.Msg) error {
		var urlMsg proto.URLFoundMsg
		if err := natsutil.ReadJSON(msg, &urlMsg); err != nil {
			return err
		}

		logrus.Debugf("Processing URL: %s", urlMsg.URL)
		normalizedURL, err := normalizeURL(urlMsg.URL)
		if err != nil {
			logrus.Errorf("Error while normalizing URL: %s", err)
			return err
		}

		// Make sure URL is valid .onion
		if !strings.Contains(normalizedURL.Host, ".onion") {
			logrus.Debugf("Url %s is not a valid hidden service", normalizedURL)
			return err
		}

		b64URI := base64.URLEncoding.EncodeToString([]byte(normalizedURL.String()))
		apiURL := fmt.Sprintf("%s/v1/resources?url=%s", apiURI, b64URI)

		var urls []proto.ResourceDto
		r, err := httpClient.JSONGet(apiURL, &urls)
		if err != nil {
			logrus.Errorf("Error while searching URL: %s", err)
			if r != nil {
				logrus.Errorf("Received status code: %d", r.StatusCode)
			}
			return err
		}

		// No matches: schedule!
		if len(urls) == 0 {
			logrus.Debugf("%s should be scheduled", normalizedURL)
			if err := natsutil.PublishJSON(nc, proto.URLTodoSubject, &proto.URLTodoMsg{URL: urlMsg.URL}); err != nil {
				return fmt.Errorf("error while publishing URL: %s", err)
			}
		} else {
			logrus.Tracef("%s should not scheduled", normalizedURL)
		}

		return nil
	}
}

func normalizeURL(u string) (*url.URL, error) {
	normalizedURL, err := purell.NormalizeURLString(u, purell.FlagsUsuallySafeGreedy|
		purell.FlagRemoveDirectoryIndex|purell.FlagRemoveFragment|purell.FlagRemoveDuplicateSlashes)
	if err != nil {
		return nil, fmt.Errorf("error while normalizing URL %s: %s", u, err)
	}

	nu, err := url.Parse(normalizedURL)
	if err != nil {
		return nil, fmt.Errorf("error while parsing URL: %s", err)
	}

	return nu, nil
}
