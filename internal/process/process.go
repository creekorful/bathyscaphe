package process

import (
	"crypto/tls"
	"fmt"
	"github.com/creekorful/trandoshan/api"
	"github.com/creekorful/trandoshan/internal/archiver/storage"
	"github.com/creekorful/trandoshan/internal/clock"
	configapi "github.com/creekorful/trandoshan/internal/configapi/client"
	"github.com/creekorful/trandoshan/internal/crawler/http"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/creekorful/trandoshan/internal/logging"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
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
	TorURIFlag       = "tor-uri"
	UserAgentFlag    = "user-agent"
	ConfigAPIURIFlag = "config-api-uri"
	StorageDirFlag   = "storage-dir"
)

type Provider interface {
	Clock() (clock.Clock, error)
	ConfigClient(keys []string) (configapi.Client, error)
	APIClient() (api.API, error)
	FastHTTPClient() (http.Client, error)
	Subscriber() (event.Subscriber, error)
	ArchiverStorage() (storage.Storage, error)
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

func (p *defaultProvider) FastHTTPClient() (http.Client, error) {
	return http.NewFastHTTPClient(&fasthttp.Client{
		// Use given TOR proxy to reach the hidden services
		Dial: fasthttpproxy.FasthttpSocksDialer(p.ctx.String(TorURIFlag)),
		// Disable SSL verification since we do not really care about this
		TLSConfig:    &tls.Config{InsecureSkipVerify: true},
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 5,
		Name:         p.ctx.String(UserAgentFlag),
	}), nil
}

func (p *defaultProvider) Subscriber() (event.Subscriber, error) {
	return event.NewSubscriber(p.ctx.String(HubURIFlag))
}

func (p *defaultProvider) ArchiverStorage() (storage.Storage, error) {
	return storage.NewLocalStorage(p.ctx.String(StorageDirFlag))
}

type SubscriberDef struct {
	Exchange string
	Queue    string
	Handler  event.Handler
}

type Process interface {
	Name() string
	FlagsNames() []string
	Provide(provider Provider) error
	Subscribers() []SubscriberDef
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

	// Add custom flags
	flags := getCustomFlags()
	for _, flag := range process.FlagsNames() {
		if customFlag, contains := flags[flag]; contains {
			app.Flags = append(app.Flags, customFlag)
		}
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

			for _, subscriberDef := range process.Subscribers() {
				if err := sub.Subscribe(subscriberDef.Exchange, subscriberDef.Queue, subscriberDef.Handler); err != nil {
					return err
				}
			}
		}

		log.Info().
			Str("ver", c.App.Version).
			Msg(fmt.Sprintf("Started %s", c.App.Name))

		// Handle graceful shutdown
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

		// Block until we receive our signal.
		<-ch

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
	flags[StorageDirFlag] = &cli.StringFlag{
		Name:     StorageDirFlag,
		Usage:    "Path to the storage directory",
		Required: true,
	}

	return flags
}
