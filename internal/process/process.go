package process

import (
	"context"
	"fmt"
	"github.com/creekorful/trandoshan/internal/cache"
	"github.com/creekorful/trandoshan/internal/clock"
	configapi "github.com/creekorful/trandoshan/internal/configapi/client"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/creekorful/trandoshan/internal/indexer/client"
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
	version = "0.9.0"
	// APIURIFlag is the api-uri flag
	APIURIFlag = "api-uri"
	// APITokenFlag is the api-token flag
	APITokenFlag = "api-token"
	// HubURIFlag is the hub-uri flag
	HubURIFlag = "hub-uri"
	// ConfigAPIURIFlag is the config-api-uri flag
	ConfigAPIURIFlag = "config-api-uri"
	// RedisURIFlag is the redis-uri flag
	RedisURIFlag = "redis-uri"
)

// Provider is the implementation provider
type Provider interface {
	// Clock return a clock implementation
	Clock() (clock.Clock, error)
	// ConfigClient return a new configured configapi.Client
	ConfigClient(keys []string) (configapi.Client, error)
	// IndexerClient return a new configured indexer client
	IndexerClient() (client.Client, error)
	// Subscriber return a new configured subscriber
	Subscriber() (event.Subscriber, error)
	// Publisher return a new configured publisher
	Publisher() (event.Publisher, error)
	// Cache return a new configured cache
	Cache() (cache.Cache, error)
	// GetValue return value for given key
	GetValue(key string) string
	// GetValue return values for given key
	GetValues(key string) []string
}

type defaultProvider struct {
	ctx *cli.Context
}

// NewDefaultProvider create a brand new default provider using given cli.Context
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

func (p *defaultProvider) IndexerClient() (client.Client, error) {
	return client.NewClient(p.ctx.String(APIURIFlag), p.ctx.String(APITokenFlag)), nil
}

func (p *defaultProvider) Subscriber() (event.Subscriber, error) {
	return event.NewSubscriber(p.ctx.String(HubURIFlag))
}

func (p *defaultProvider) Publisher() (event.Publisher, error) {
	return event.NewPublisher(p.ctx.String(HubURIFlag))
}

func (p *defaultProvider) Cache() (cache.Cache, error) {
	return cache.NewRedisCache(p.ctx.String(RedisURIFlag))
}

func (p *defaultProvider) GetValue(key string) string {
	return p.ctx.String(key)
}

func (p *defaultProvider) GetValues(key string) []string {
	return p.ctx.StringSlice(key)
}

// SubscriberDef is the subscriber definition
type SubscriberDef struct {
	Exchange string
	Queue    string
	Handler  event.Handler
}

// Process is a component of Trandoshan
type Process interface {
	Name() string
	CommonFlags() []string
	CustomFlags() []cli.Flag
	Initialize(provider Provider) error
	Subscribers() []SubscriberDef
	HTTPHandler(provider Provider) http.Handler
}

// MakeApp return cli.App corresponding for given Process
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
		if err := process.Initialize(provider); err != nil {
			log.Err(err).Msg("error while initializing app")
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
					log.Err(err).
						Str("exchange", subscriberDef.Exchange).
						Str("queue", subscriberDef.Queue).
						Msg("error while subscribing")
					return err
				}
			}
		}

		var srv *http.Server

		// Expose HTTP API if any
		if h := process.HTTPHandler(provider); h != nil {
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
	flags[RedisURIFlag] = &cli.StringFlag{
		Name:     RedisURIFlag,
		Usage:    "URI to the Redis server",
		Required: true,
	}

	return flags
}
