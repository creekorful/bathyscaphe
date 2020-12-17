package api

import (
	"github.com/creekorful/trandoshan/api"
	"github.com/creekorful/trandoshan/internal/api/database"
	"github.com/creekorful/trandoshan/internal/duration"
	"github.com/creekorful/trandoshan/internal/messaging"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"time"
)

type service interface {
	searchResources(params *database.ResSearchParams) ([]api.ResourceDto, int64, error)
	addResource(res api.ResourceDto) (api.ResourceDto, error)
	scheduleURL(url string) error
	close()
}

type svc struct {
	db           database.Database
	pub          messaging.Publisher
	refreshDelay time.Duration
}

func newService(c *cli.Context) (service, error) {
	// Connect to the messaging server
	pub, err := messaging.NewPublisher(c.String("hub-uri"))
	if err != nil {
		log.Err(err).Str("uri", c.String("hub-uri")).Msg("Error while connecting to hub server")
		return nil, err
	}

	// Create Elasticsearch client
	db, err := database.NewElasticDB(c.String("elasticsearch-uri"))
	if err != nil {
		log.Err(err).Msg("Error while connecting to the database")
		return nil, err
	}

	refreshDelay := duration.ParseDuration(c.String("refresh-delay"))

	return &svc{
		db:           db,
		pub:          pub,
		refreshDelay: refreshDelay,
	}, nil
}

func (s *svc) searchResources(params *database.ResSearchParams) ([]api.ResourceDto, int64, error) {
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

func (s *svc) addResource(res api.ResourceDto) (api.ResourceDto, error) {
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

func (s *svc) scheduleURL(url string) error {
	// Publish the URL
	if err := s.pub.PublishMsg(&messaging.URLFoundMsg{URL: url}); err != nil {
		log.Err(err).Msg("Unable to publish URL")
		return err
	}

	log.Debug().Str("url", url).Msg("Successfully published URL")
	return nil
}

func (s *svc) close() {
	s.pub.Close()
}
