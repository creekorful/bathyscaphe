package process

//go:generate mockgen -destination=../process_mock/process_mock.go -package=process_mock . Provider

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/creekorful/trandoshan/internal/cache"
	"github.com/creekorful/trandoshan/internal/clock"
	configapi "github.com/creekorful/trandoshan/internal/configapi/client"
	"github.com/creekorful/trandoshan/internal/event"
	chttp "github.com/creekorful/trandoshan/internal/http"
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

const (
	version = "0.10.0"
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
	// TorURIFlag is the tor-uri flag
	TorURIFlag = "tor-uri"
	// UserAgentFlag is the user-agent flag
	UserAgentFlag = "user-agent"
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

func (p *defaultProvider) Subscriber() (event.Subscriber, error) {
	return event.NewSubscriber(p.ctx.String(HubURIFlag))
}

func (p *defaultProvider) Publisher() (event.Publisher, error) {
	return event.NewPublisher(p.ctx.String(HubURIFlag))
}

func (p *defaultProvider) Cache(keyPrefix string) (cache.Cache, error) {
	return cache.NewRedisCache(p.ctx.String(RedisURIFlag), keyPrefix)
}

func (p *defaultProvider) HTTPClient() (chttp.Client, error) {
	return chttp.NewFastHTTPClient(&fasthttp.Client{
		// Use given TOR proxy to reach the hidden services
		Dial: fasthttpproxy.FasthttpSocksDialer(p.ctx.String(TorURIFlag)),
		// Disable SSL verification since we do not really care about this
		TLSConfig:    &tls.Config{InsecureSkipVerify: true},
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 5,
		Name:         p.ctx.String(UserAgentFlag),
	}), nil
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
	HTTPHandler() http.Handler
}

// MakeApp return cli.App corresponding for given Process
func MakeApp(process Process) *cli.App {
	app := &cli.App{
		Name:    fmt.Sprintf("tdsh-%s", process.Name()),
		Version: version,
		Usage:   fmt.Sprintf("Trandoshan %s component", process.Name()),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "log-level",
				Usage: "Set the application log level",
				Value: "info",
			},
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
	flags[TorURIFlag] = &cli.StringFlag{
		Name:     TorURIFlag,
		Usage:    "URI to the TOR SOCKS proxy",
		Required: true,
	}
	flags[UserAgentFlag] = &cli.StringFlag{
		Name:  UserAgentFlag,
		Usage: "User agent to use",
		Value: "Mozilla/5.0 (Windows NT 10.0; rv:68.0) Gecko/20100101 Firefox/68.0",
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
