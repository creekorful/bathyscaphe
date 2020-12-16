package trandoshanctl

import (
	"fmt"
	"github.com/creekorful/trandoshan/internal/logging"
	"github.com/creekorful/trandoshan/internal/util"
	"github.com/olekukonko/tablewriter"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"os"
	"time"
)

// GetApp returns the Trandoshan CLI app
func GetApp() *cli.App {
	apiFlag := util.GetAPIURIFlag()
	apiFlag.Value = "http://localhost:15005"
	apiFlag.Required = false

	return &cli.App{
		Name:    "trandoshanctl",
		Version: "0.6.0",
		Usage:   "Trandoshan CLI",
		Flags: []cli.Flag{
			logging.GetLogFlag(),
			apiFlag,
			util.GetAPITokenFlag(),
		},
		Commands: []*cli.Command{
			{
				Name:      "schedule",
				Usage:     "Schedule crawling for given URL",
				Action:    schedule,
				ArgsUsage: "URL",
			},
			{
				Name:      "search",
				Usage:     "Search for specific resources",
				ArgsUsage: "keyword",
				Action:    search,
			},
		},
		Before: before,
	}
}

func before(ctx *cli.Context) error {
	logging.ConfigureLogger(ctx)
	return nil
}

func schedule(c *cli.Context) error {
	if c.NArg() == 0 {
		return fmt.Errorf("missing argument URL")
	}

	url := c.Args().First()

	// Create the API client
	apiClient := util.GetAPIClient(c)

	if err := apiClient.ScheduleURL(url); err != nil {
		log.Err(err).Str("url", url).Msg("Unable to schedule crawling for URL")
		return err
	}

	log.Info().Str("url", url).Msg("Successfully schedule crawling")

	return nil
}

func search(c *cli.Context) error {
	keyword := c.Args().First()

	// Create the API client
	apiClient := util.GetAPIClient(c)

	res, count, err := apiClient.SearchResources("", keyword, time.Time{}, time.Time{}, 1, 10)
	if err != nil {
		log.Err(err).Str("keyword", keyword).Msg("Unable to search resources")
		return err
	}

	if len(res) == 0 {
		fmt.Println("No resources crawled (yet).")
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Time", "URL", "Title"})

	for _, v := range res {
		table.Append([]string{v.Time.Format(time.RFC822), shortenURL(v.URL), v.Title})
	}
	table.Render()

	fmt.Printf("Total: %d\n", count)

	return nil
}

func shortenURL(url string) string {
	if len(url) > 125 {
		url := url[0:125]
		return url + "..."
	}

	return url
}
