package service

import (
	"fmt"
	"github.com/creekorful/trandoshan/api"
	"github.com/creekorful/trandoshan/internal/api/database"
	"github.com/creekorful/trandoshan/internal/duration"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"time"
)

// Service is the functionality the API provide
type Service interface {
	SearchResources(params *database.ResSearchParams) ([]api.ResourceDto, int64, error)
	AddResource(res api.ResourceDto) (api.ResourceDto, error)
	ScheduleURL(url string) error
	Close()
}

type svc struct {
	db           database.Database
	pub          event.Publisher
	refreshDelay time.Duration
}

// New create a new Service instance
func New(c *cli.Context) (Service, error) {
	// Connect to the messaging server
	pub, err := event.NewPublisher(c.String("hub-uri"))
	if err != nil {
		return nil, fmt.Errorf("error while connecting to hub server: %s", err)
	}

	// Create Elasticsearch client
	db, err := database.NewElasticDB(c.String("elasticsearch-uri"))
	if err != nil {
		return nil, fmt.Errorf("error while connecting to the database: %s", err)
	}

	refreshDelay := duration.ParseDuration(c.String("refresh-delay"))

	return &svc{
		db:           db,
		pub:          pub,
		refreshDelay: refreshDelay,
	}, nil
}

func (s *svc) SearchResources(params *database.ResSearchParams) ([]api.ResourceDto, int64, error) {
	totalCount, err := s.db.CountResources(params)
	if err != nil {
		log.Err(err).Msg("Error while counting on ES")
		return nil, 0, err
	}

	res, err := s.db.SearchResources(params)
	if err != nil {
		log.Err(err).Msg("Error while searching on ES")
		return nil, 0, err
	}

	var resources []api.ResourceDto
	for _, r := range res {
		resources = append(resources, api.ResourceDto{
			URL:   r.URL,
			Body:  r.Body,
			Title: r.Title,
			Time:  r.Time,
		})
	}

	return resources, totalCount, nil
}

func (s *svc) AddResource(res api.ResourceDto) (api.ResourceDto, error) {
	log.Debug().Str("url", res.URL).Msg("Saving resource")

	// Hacky stuff to prevent from adding 'duplicate resource'
	// the thing is: even with the scheduler preventing from crawling 'duplicates' URL by adding a refresh period
	// and checking if the resource is not already indexed,  this implementation may not work if the URLs was published
	// before the resource is saved. And this happen a LOT of time.
	// therefore the best thing to do is to make the API check if the resource should **really** be added by checking if
	// it isn't present on the database. This may sounds hacky, but it's the best solution i've come up at this time.
	endDate := time.Time{}
	if s.refreshDelay != -1 {
		endDate = time.Now().Add(-s.refreshDelay)
	}

	count, err := s.db.CountResources(&database.ResSearchParams{
		URL:        res.URL,
		EndDate:    endDate,
		PageSize:   1,
		PageNumber: 1,
	})
	if err != nil {
		log.Err(err).Msg("error while searching for resource")
		return api.ResourceDto{}, nil
	}

	if count > 0 {
		// Not an error
		log.Debug().Str("url", res.URL).Msg("Skipping duplicate resource")
		return res, nil
	}

	// Create Elasticsearch document
	doc := database.ResourceIdx{
		URL:         res.URL,
		Body:        res.Body,
		Time:        res.Time,
		Title:       res.Title,
		Meta:        res.Meta,
		Description: res.Description,
		Headers:     res.Headers,
	}

	if err := s.db.AddResource(doc); err != nil {
		log.Err(err).Msg("Error while adding resource")
		return api.ResourceDto{}, err
	}

	log.Debug().Str("url", res.URL).Msg("Successfully saved resource")
	return res, nil
}

func (s *svc) ScheduleURL(url string) error {
	// Publish the URL
	if err := s.pub.Publish(&event.FoundURLEvent{URL: url}); err != nil {
		log.Err(err).Msg("Unable to publish URL")
		return err
	}

	log.Debug().Str("url", url).Msg("Successfully published URL")
	return nil
}

func (s *svc) Close() {
	s.pub.Close()
}
