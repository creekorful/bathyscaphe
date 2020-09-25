package api

import (
	"github.com/creekorful/trandoshan/api"
	"github.com/creekorful/trandoshan/internal/database"
	"github.com/creekorful/trandoshan/internal/messaging"
	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"golang.org/x/crypto/bcrypt"
	"net/http"
)

type service interface {
	searchResources(params *database.ResSearchParams) ([]api.ResourceDto, int64, error)
	addResource(res api.ResourceDto) (api.ResourceDto, error)
	scheduleURL(url string) error
	authenticate(credentials api.CredentialsDto) (string, error)
	close()
}

type svc struct {
	users      map[string][]byte
	signingKey []byte
	db         database.Database
	pub        messaging.Publisher
}

func newService(c *cli.Context, signingKey []byte) (service, error) {
	// Connect to the NATS server
	pub, err := messaging.NewPublisher(c.String("nats-uri"))
	if err != nil {
		log.Err(err).Str("uri", c.String("nats-uri")).Msg("Error while connecting to NATS server")
		return nil, err
	}

	// Create Elasticsearch client
	db, err := database.NewElasticDB(c.String("elasticsearch-uri"))
	if err != nil {
		log.Err(err).Msg("Error while connecting to the database")
		return nil, err
	}

	return &svc{
		db:         db,
		users:      map[string][]byte{},
		signingKey: signingKey,
		pub:        pub,
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

	// Create Elasticsearch document
	doc := database.ResourceIdx{
		URL:   res.URL,
		Body:  res.Body,
		Title: res.Title,
		Time:  res.Time,
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

func (s *svc) authenticate(credentials api.CredentialsDto) (string, error) {
	if credentials.Username == "" || credentials.Password == "" {
		log.Warn().Msg("Invalid credentials supplied")
		return "", echo.NewHTTPError(http.StatusUnprocessableEntity)
	}

	// Try to find the user
	pass, exists := s.users[credentials.Username]
	if !exists {
		log.Warn().Str("username", credentials.Username).Msg("No user found")
		return "", echo.NewHTTPError(http.StatusUnprocessableEntity)
	}

	// Validate provided password
	if err := bcrypt.CompareHashAndPassword(pass, []byte(credentials.Password)); err != nil {
		log.Warn().Str("username", credentials.Username).Msg("Invalid password")
		return "", echo.NewHTTPError(http.StatusUnauthorized)
	}

	log.Debug().Str("username", credentials.Username).Msg("Successfully logged-in")

	// Build JWT token
	claims := jwt.MapClaims{
		"username": credentials.Username,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign JWT token
	return token.SignedString(s.signingKey)
}

func (s *svc) close() {
	s.pub.Close()
}
