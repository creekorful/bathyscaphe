package process

//go:generate mockgen -destination=../process_mock/process_mock.go -package=process_mock . Provider

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/darkspot-org/bathyscaphe/internal/cache"
	"github.com/darkspot-org/bathyscaphe/internal/clock"
	configapi "github.com/darkspot-org/bathyscaphe/internal/configapi/client"
	"github.com/darkspot-org/bathyscaphe/internal/event"
	chttp "github.com/darkspot-org/bathyscaphe/internal/http"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Feature represent a process feature
type Feature int

const (
	version = "1.0.0-rc2"

	// EventFeature is the feature to plug the process to the event server
	EventFeature Feature = iota
	// ConfigFeature is the feature to plug the process to the ConfigAPI framework
	ConfigFeature
	// CacheFeature is the feature to plug the process to the cache server
	CacheFeature
	// CrawlingFeature is the feature to plug the process with a tor-compatible HTTP client
	CrawlingFeature

	// EventPrefetchFlag is the prefetch count for the event subscriber
	EventPrefetchFlag = "event-prefetch"

	eventURIFlag     = "event-srv"
	configAPIURIFlag = "config-api"
	cacheSRVFlag     = "cache-srv"
	torURIFlag       = "tor-proxy"
	userAgentFlag    = "user-agent"
)

// Provider is the implementation provider
type Provider interface {
	// Clock return a clock implementation
	Clock() (clock.Clock, error)
	// ConfigClient return a new configured configapi.Client
	ConfigClient(keys []string) (configapi.Client, error)
	// Subscriber return a new configured subscriber
	Subscriber() (event.Subscriber, error)
	// Publisher return a new configured publisher
	Publisher() (event.Publisher, error)
	// Cache return a new configured cache
	Cache(keyPrefix string) (cache.Cache, error)
	// HTTPClient return a new configured http client
	HTTPClient() (chttp.Client, error)
	// GetStrValue return string value for given key
	GetStrValue(key string) string
	// GetStrValues return string slice for given key
	GetStrValues(key string) []string
	// GetIntValue return int value for given key
	GetIntValue(key string) int
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

	return configapi.NewConfigClient(p.ctx.String(configAPIURIFlag), sub, keys)
}

func (p *defaultProvider) Subscriber() (event.Subscriber, error) {
	return event.NewSubscriber(p.ctx.String(eventURIFlag), p.ctx.Int(EventPrefetchFlag))
}

func (p *defaultProvider) Publisher() (event.Publisher, error) {
	return event.NewPublisher(p.ctx.String(eventURIFlag))
}

func (p *defaultProvider) Cache(keyPrefix string) (cache.Cache, error) {
	return cache.NewRedisCache(p.ctx.String(cacheSRVFlag), keyPrefix)
}

func (p *defaultProvider) HTTPClient() (chttp.Client, error) {
	return chttp.NewFastHTTPClient(&fasthttp.Client{
		// Use given TOR proxy to reach the hidden services
		Dial: fasthttpproxy.FasthttpSocksDialer(p.ctx.String(torURIFlag)),
		// Disable SSL verification since we do not really care about this
		TLSConfig:    &tls.Config{InsecureSkipVerify: true},
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 5,
		Name:         p.ctx.String(userAgentFlag),
	}), nil
}

func (p *defaultProvider) GetStrValue(key string) string {
	return p.ctx.String(key)
}

func (p *defaultProvider) GetStrValues(key string) []string {
	return p.ctx.StringSlice(key)
}

func (p *defaultProvider) GetIntValue(key string) int {
	return p.ctx.Int(key)
}

// SubscriberDef is the subscriber definition
type SubscriberDef struct {
	Exchange string
	Queue    string
	Handler  event.Handler
}

// Process is a component of Bathyscaphe
type Process interface {
	Name() string
	Description() string
	Features() []Feature
	CustomFlags() []cli.Flag
	Initialize(provider Provider) error
	Subscribers() []SubscriberDef
	HTTPHandler() http.Handler
}

// MakeApp return cli.App corresponding for given Process
func MakeApp(process Process) *cli.App {
	app := &cli.App{
		Name:        fmt.Sprintf("bs-%s", process.Name()),
		Version:     version,
		Usage:       fmt.Sprintf("Bathyscaphe %s component", process.Name()),
		Description: process.Description(),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "log-level",
				Usage: "Set the application log level",
				Value: "info",
			},
		},
		Authors: []*cli.Author{
			{
				Name:  "Aloïs Micard",
				Email: "alois@micard.lu",
			},
		},
		Action: execute(process),
	}

	// Add features flags
	featureFlags := getFeaturesFlags()
	for _, feature := range process.Features() {
		if values, exist := featureFlags[feature]; exist {
			app.Flags = append(app.Flags, values...)
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
		configureLogger(c)

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

func getFeaturesFlags() map[Feature][]cli.Flag {
	flags := map[Feature][]cli.Flag{}

	flags[EventFeature] = []cli.Flag{
		&cli.StringFlag{
			Name:     eventURIFlag,
			Usage:    "URI to the event server",
			Required: true,
		},
		&cli.IntFlag{
			Name:  EventPrefetchFlag,
			Usage: "Prefetch for the event subscriber",
			Value: 1,
		},
	}

	flags[ConfigFeature] = []cli.Flag{
		&cli.StringFlag{
			Name:     configAPIURIFlag,
			Usage:    "URI to the ConfigAPI server",
			Required: true,
		},
	}

	flags[CacheFeature] = []cli.Flag{
		&cli.StringFlag{
			Name:     cacheSRVFlag,
			Usage:    "URI to the cache server",
			Required: true,
		},
	}

	flags[CrawlingFeature] = []cli.Flag{
		&cli.StringFlag{
			Name:     torURIFlag,
			Usage:    "URI to the TOR SOCKS proxy",
			Required: true,
		},
		&cli.StringFlag{
			Name:  userAgentFlag,
			Usage: "User agent to use",
			Value: "Mozilla/5.0 (Windows NT 10.0; rv:68.0) Gecko/20100101 Firefox/68.0",
		},
	}

	return flags
}

func configureLogger(ctx *cli.Context) {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Set application log level
	if lvl, err := zerolog.ParseLevel(ctx.String("log-level")); err == nil {
		zerolog.SetGlobalLevel(lvl)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	log.Debug().Stringer("lvl", zerolog.GlobalLevel()).Msg("Setting log level")
}
