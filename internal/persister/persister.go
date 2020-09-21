package persister

import (
	"fmt"
	"github.com/creekorful/trandoshan/internal/util/http"
	"github.com/creekorful/trandoshan/internal/util/logging"
	natsutil "github.com/creekorful/trandoshan/internal/util/nats"
	"github.com/creekorful/trandoshan/pkg/proto"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

// GetApp return the persister app
func GetApp() *cli.App {
	return &cli.App{
		Name:    "tdsh-persister",
		Version: "0.3.0",
		Usage:   "Trandoshan persister process",
		Flags: []cli.Flag{
			logging.GetLogFlag(),
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
	logging.ConfigureLogger(ctx)

	log.Info().Str("ver", ctx.App.Version).Msg("Starting tdsh-persister")

	log.Debug().Str("uri", ctx.String("nats-uri")).Msg("Using NATS server")
	log.Debug().Str("uri", ctx.String("api-uri")).Msg("Using API server")

	// Create the HTTP client
	httpClient := &http.Client{}

	// Create the NATS subscriber
	sub, err := natsutil.NewSubscriber(ctx.String("nats-uri"))
	if err != nil {
		return err
	}
	defer sub.Close()

	log.Info().Msg("Successfully initialized tdsh-persister. Waiting for resources")

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

		log.Debug().Str("url", resMsg.URL).Msg("Processing resource")

		url := fmt.Sprintf("%s/v1/resources", apiURI)
		r, err := httpClient.JSONPost(url, &proto.ResourceDto{
			URL:  resMsg.URL,
			Body: resMsg.Body,
		}, nil)

		if err != nil || r.StatusCode != http.StatusCreated {
			log.Err(err).Msg("Error while sending resource to the API")
			return err
		}

		log.Debug().Str("url", resMsg.URL).Msg("Successfully processed resource")

		return nil
	}
}
