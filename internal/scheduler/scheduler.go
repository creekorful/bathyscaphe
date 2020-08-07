package scheduler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/purell"
	"github.com/creekorful/trandoshan/internal/api"
	"github.com/creekorful/trandoshan/internal/log"
	"github.com/creekorful/trandoshan/internal/natsutil"
	"github.com/creekorful/trandoshan/pkg/proto"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"net/http"
	"net/url"
	"strings"
)

// GetApp return the scheduler app
func GetApp() *cli.App {
	return &cli.App{
		Name:    "trandoshan-scheduler",
		Version: "0.0.1",
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

	logrus.Infof("Starting trandoshan-scheduler v%s", ctx.App.Version)

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

	logrus.Info("Successfully initialized trandoshan-scheduler. Waiting for URLs")

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

		// Normalized received URL
		normalizedURL, err := purell.NormalizeURLString(urlMsg.URL, purell.FlagsUsuallySafeGreedy|
			purell.FlagRemoveDirectoryIndex|purell.FlagRemoveFragment|purell.FlagRemoveDuplicateSlashes)

		if err != nil {
			return fmt.Errorf("error while normalizing URL %s: %s", urlMsg.URL, err)
		}

		logrus.Tracef("Normalized URL: %s", normalizedURL)

		// Make sure URL is valid .onion
		u, err := url.Parse(normalizedURL)
		if err != nil {
			logrus.Errorf("Error while parsing URL: %s", err)
			return err
		}

		if !strings.Contains(u.Host, ".onion") {
			logrus.Debugf("Url %s is not a valid hidden service", normalizedURL)
			return err
		}

		b64URI := base64.URLEncoding.EncodeToString([]byte(normalizedURL))
		apiURL := fmt.Sprintf("%s/v1/resources?url=%s", apiURI, b64URI)
		logrus.Tracef("Using API URL: %s", apiURL)

		resp, err := httpClient.Get(apiURL)
		if err != nil || resp.StatusCode != http.StatusOK {
			logrus.Errorf("Error while searching URL: %s", err)
			logrus.Errorf("Received status code: %d", resp.StatusCode)
			return err
		}
		defer resp.Body.Close()

		var urls []api.ResourceDto
		if err := json.NewDecoder(resp.Body).Decode(&urls); err != nil {
			logrus.Errorf("Error while un-marshaling urls: %s", err)
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
