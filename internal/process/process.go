package process

import (
	"context"
	"fmt"
	"github.com/creekorful/trandoshan/api"
	"github.com/creekorful/trandoshan/internal/clock"
	configapi "github.com/creekorful/trandoshan/internal/configapi/client"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/creekorful/trandoshan/internal/logging"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	version          = "0.7.0"
	APIURIFlag       = "api-uri"
	APITokenFlag     = "api-token"
	HubURIFlag       = "hub-uri"
	ConfigAPIURIFlag = "config-api-uri"
)

type Provider interface {
	Clock() (clock.Clock, error)
	ConfigClient(keys []string) (configapi.Client, error)
	APIClient() (api.API, error)
	Subscriber() (event.Subscriber, error)
	GetValue(key string) string
	GetValues(key string) []string
}

type defaultProvider struct {
	ctx *cli.Context
}

func NewDefaultProvider(ctx *cli.Context) Provider {
	return &defaultProvider{ctx: ctx}
}

func (p *defaultProvider) Clock() (clock.Clock, error) {
	return &clock.SystemClock{}, nil
}

func (p *defaultProvider) ConfigClient(keys []string) (configapi.Client, error) {
	sub, err := p.Subscriber()
	if err != nil {
		return nil, err
	}

	return configapi.NewConfigClient(p.ctx.String(ConfigAPIURIFlag), sub, keys)
}

func (p *defaultProvider) APIClient() (api.API, error) {
	return api.NewClient(p.ctx.String(APIURIFlag), p.ctx.String(APITokenFlag)), nil
}

func (p *defaultProvider) Subscriber() (event.Subscriber, error) {
	return event.NewSubscriber(p.ctx.String(HubURIFlag))
}

func (p *defaultProvider) GetValue(key string) string {
	return p.ctx.String(key)
}

func (p *defaultProvider) GetValues(key string) []string {
	return p.ctx.StringSlice(key)
}

type SubscriberDef struct {
	Exchange string
	Queue    string
	Handler  event.Handler
}

type Process interface {
	Name() string
	CommonFlags() []string
	CustomFlags() []cli.Flag
	Provide(provider Provider) error
	Subscribers() []SubscriberDef
	HTTPHandler() http.Handler
}

func MakeApp(process Process) *cli.App {
	app := &cli.App{
		Name:    fmt.Sprintf("tdsh-%s", process.Name()),
		Version: version,
		Usage:   fmt.Sprintf("Trandoshan %s component", process.Name()),
		Flags: []cli.Flag{
			logging.GetLogFlag(),
		},
		Action: execute(process),
	}

	// Add common flags
	flags := getCustomFlags()
	for _, flag := range process.CommonFlags() {
		if customFlag, contains := flags[flag]; contains {
			app.Flags = append(app.Flags, customFlag)
		}
	}

	// Add custom flags
	for _, flag := range process.CustomFlags() {
		app.Flags = append(app.Flags, flag)
	}

	return app
}

func execute(process Process) cli.ActionFunc {
	return func(c *cli.Context) error {
		provider := NewDefaultProvider(c)

		// Common setup
		logging.ConfigureLogger(c)

		// Custom setup
		if err := process.Provide(provider); err != nil {
			return err
		}

		// Create subscribers if any
		if len(process.Subscribers()) > 0 {
			sub, err := provider.Subscriber()
			if err != nil {
				return err
			}
			// TODO sub.Close()

			for _, subscriberDef := range process.Subscribers() {
				if err := sub.Subscribe(subscriberDef.Exchange, subscriberDef.Queue, subscriberDef.Handler); err != nil {
					return err
				}
			}
		}

		var srv *http.Server

		// Expose HTTP API if any
		if h := process.HTTPHandler(); h != nil {
			srv = &http.Server{
				Addr: "0.0.0.0:8080",
				// Good practice to set timeouts to avoid Slowloris attacks.
				WriteTimeout: time.Second * 15,
				ReadTimeout:  time.Second * 15,
				IdleTimeout:  time.Second * 60,
				Handler:      h, // Pass our instance of gorilla/mux in.
			}

			go func() {
				_ = srv.ListenAndServe()
			}()
		}

		log.Info().
			Str("ver", c.App.Version).
			Msg(fmt.Sprintf("Started %s", c.App.Name))

		// Handle graceful shutdown
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

		// Block until we receive our signal.
		<-ch

		// Close HTTP API if any
		if srv != nil {
			_ = srv.Shutdown(context.Background())
		}

		// Connections are deferred here

		return nil
	}
}

func getCustomFlags() map[string]cli.Flag {
	flags := map[string]cli.Flag{}

	flags[HubURIFlag] = &cli.StringFlag{
		Name:     HubURIFlag,
		Usage:    "URI to the hub (event) server",
		Required: true,
	}
	flags[APIURIFlag] = &cli.StringFlag{
		Name:     APIURIFlag,
		Usage:    "URI to the API server",
		Required: true,
	}
	flags[APITokenFlag] = &cli.StringFlag{
		Name:     APITokenFlag,
		Usage:    "Token to use to authenticate against the API",
		Required: true,
	}
	flags[ConfigAPIURIFlag] = &cli.StringFlag{
		Name:     ConfigAPIURIFlag,
		Usage:    "URI to the ConfigAPI server",
		Required: true,
	}

	return flags
}
