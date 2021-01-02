package indexer

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/PuerkitoBio/purell"
	configapi "github.com/creekorful/trandoshan/internal/configapi/client"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/creekorful/trandoshan/internal/indexer/auth"
	"github.com/creekorful/trandoshan/internal/indexer/client"
	"github.com/creekorful/trandoshan/internal/indexer/database"
	"github.com/creekorful/trandoshan/internal/process"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"mvdan.cc/xurls/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	defaultPaginationSize = 50
	maxPaginationSize     = 100

	errHostnameNotAllowed = fmt.Errorf("hostname is not allowed")
	errAlreadyIndexed     = fmt.Errorf("resource is already indexed")
)

// State represent the application state
type State struct {
	db           database.Database
	pub          event.Publisher
	configClient configapi.Client
}

// Name return the process name
func (state *State) Name() string {
	return "indexer"
}

// CommonFlags return process common flags
func (state *State) CommonFlags() []string {
	return []string{process.HubURIFlag, process.ConfigAPIURIFlag}
}

// CustomFlags return process custom flags
func (state *State) CustomFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "elasticsearch-uri",
			Usage:    "URI to the Elasticsearch server",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "signing-key",
			Usage:    "Signing key for the JWT token",
			Required: true,
		},
	}
}

// Initialize the process
func (state *State) Initialize(provider process.Provider) error {
	db, err := database.NewElasticDB(provider.GetValue("elasticsearch-uri"))
	if err != nil {
		return err
	}
	state.db = db

	pub, err := provider.Subscriber()
	if err != nil {
		return err
	}
	state.pub = pub

	configClient, err := provider.ConfigClient([]string{configapi.RefreshDelayKey, configapi.ForbiddenHostnamesKey})
	if err != nil {
		return err
	}
	state.configClient = configClient

	return nil
}

// Subscribers return the process subscribers
func (state *State) Subscribers() []process.SubscriberDef {
	return []process.SubscriberDef{
		{Exchange: event.NewResourceExchange, Queue: "indexingQueue", Handler: state.handleNewResourceEvent},
	}
}

// HTTPHandler returns the HTTP API the process expose
func (state *State) HTTPHandler(provider process.Provider) http.Handler {
	r := mux.NewRouter()

	signingKey := []byte(provider.GetValue("signing-key"))
	authMiddleware := auth.NewMiddleware(signingKey)
	r.Use(authMiddleware.Middleware())

	r.HandleFunc("/v1/resources", state.searchResources).Methods(http.MethodGet)
	r.HandleFunc("/v1/urls", state.scheduleURL).Methods(http.MethodPost)

	return r
}

func (state *State) searchResources(w http.ResponseWriter, r *http.Request) {
	searchParams, err := getSearchParams(r)
	if err != nil {
		log.Err(err).Msg("error while getting search params")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	totalCount, err := state.db.CountResources(searchParams)
	if err != nil {
		log.Err(err).Msg("error while counting on ES")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	res, err := state.db.SearchResources(searchParams)
	if err != nil {
		log.Err(err).Msg("error while searching on ES")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var resources []client.ResourceDto
	for _, r := range res {
		resources = append(resources, client.ResourceDto{
			URL:   r.URL,
			Body:  r.Body,
			Title: r.Title,
			Time:  r.Time,
		})
	}

	// Write pagination headers
	writePagination(w, searchParams, totalCount)

	// Write body
	writeJSON(w, resources)
}

func (state *State) scheduleURL(w http.ResponseWriter, r *http.Request) {
	var url string
	if err := json.NewDecoder(r.Body).Decode(&url); err != nil {
		log.Warn().Str("err", err.Error()).Msg("error while decoding request body")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	if err := state.pub.PublishEvent(&event.FoundURLEvent{URL: url}); err != nil {
		log.Err(err).Msg("unable to schedule URL")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Info().Str("url", url).Msg("successfully scheduled URL")
}

func (state *State) handleNewResourceEvent(subscriber event.Subscriber, msg event.RawMessage) error {
	var evt event.NewResourceEvent
	if err := subscriber.Read(&msg, &evt); err != nil {
		return err
	}

	// Extract & process resource
	resDto, urls, err := extractResource(evt)
	if err != nil {
		return fmt.Errorf("error while extracting resource: %s", err)
	}

	// Lowercase headers
	resDto.Headers = map[string]string{}
	for key, value := range evt.Headers {
		resDto.Headers[strings.ToLower(key)] = value
	}

	// Save resource
	if _, err := state.addResource(resDto); err != nil {
		return err
	}

	// Finally push found URLs
	publishedURLS := map[string]string{}
	for _, u := range urls {
		if _, exist := publishedURLS[u]; exist {
			continue
		}

		// make sure url should be published
		count, err := state.countResource(u)
		if err != nil {
			continue
		}
		if count > 0 {
			continue
		}

		log.Trace().
			Str("url", u).
			Msg("Publishing found URL")

		if err := subscriber.PublishEvent(&event.FoundURLEvent{URL: u}); err != nil {
			log.Warn().
				Str("url", u).
				Str("err", err.Error()).
				Msg("Error while publishing URL")
		}

		publishedURLS[u] = u
	}

	return nil
}

func (state *State) addResource(res client.ResourceDto) (client.ResourceDto, error) {
	forbiddenHostnames, err := state.configClient.GetForbiddenHostnames()
	if err != nil {
		return client.ResourceDto{}, err
	}

	u, err := url.Parse(res.URL)
	if err != nil {
		return client.ResourceDto{}, err
	}

	for _, hostname := range forbiddenHostnames {
		if strings.Contains(u.Hostname(), hostname.Hostname) {
			return client.ResourceDto{}, errHostnameNotAllowed
		}
	}

	count, err := state.countResource(res.URL)
	if err != nil {
		return client.ResourceDto{}, err
	}
	if count > 0 {
		return client.ResourceDto{}, errAlreadyIndexed
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

	if err := state.db.AddResource(doc); err != nil {
		return client.ResourceDto{}, err
	}

	// Publish linked event
	if err := state.pub.PublishEvent(&event.NewIndexEvent{
		URL:         res.URL,
		Body:        res.Body,
		Time:        res.Time,
		Title:       res.Title,
		Meta:        res.Meta,
		Description: res.Description,
		Headers:     res.Headers,
	}); err != nil {
		return client.ResourceDto{}, err
	}

	log.Info().Str("url", res.URL).Msg("Successfully saved resource")

	return res, nil
}

// Hacky stuff to prevent from adding 'duplicate resource'
// the thing is: even with the scheduler preventing from crawling 'duplicates' URL by adding a refresh period
// and checking if the resource is not already indexed,  this implementation may not work if the URLs was published
// before the resource is saved. And this happen a LOT of time.
// therefore the best thing to do is to make the API check if the resource should **really** be added by checking if
// it isn't present on the database. This may sounds hacky, but it's the best solution i've come up at this time.
func (state *State) countResource(URL string) (int64, error) {
	endDate := time.Time{}
	if refreshDelay, err := state.configClient.GetRefreshDelay(); err == nil {
		if refreshDelay.Delay != -1 {
			endDate = time.Now().Add(-refreshDelay.Delay)
		}
	}

	count, err := state.db.CountResources(&client.ResSearchParams{
		URL:        URL,
		EndDate:    endDate,
		PageSize:   1,
		PageNumber: 1,
	})
	if err != nil {
		return -1, err
	}

	return count, nil
}

func getSearchParams(r *http.Request) (*client.ResSearchParams, error) {
	params := &client.ResSearchParams{}

	if param := r.URL.Query()["keyword"]; len(param) == 1 {
		params.Keyword = param[0]
	}

	if param := r.URL.Query()["with-body"]; len(param) == 1 {
		params.WithBody = param[0] == "true"
	}

	// extract raw query params (unescaped to keep + sign when parsing date)
	rawQueryParams := getRawQueryParam(r.URL.RawQuery)

	if val, exist := rawQueryParams["start-date"]; exist {
		d, err := time.Parse(time.RFC3339, val)
		if err == nil {
			params.StartDate = d
		} else {
			return nil, err
		}
	}

	if val, exist := rawQueryParams["end-date"]; exist {
		d, err := time.Parse(time.RFC3339, val)
		if err == nil {
			params.EndDate = d
		} else {
			return nil, err
		}
	}

	// base64decode the URL
	if param := r.URL.Query()["url"]; len(param) == 1 {
		b, err := base64.URLEncoding.DecodeString(param[0])
		if err != nil {
			return nil, err
		}
		params.URL = string(b)
	}

	// Acquire pagination
	page, size := getPagination(r)
	params.PageNumber = page
	params.PageSize = size

	return params, nil
}

func writePagination(w http.ResponseWriter, searchParams *client.ResSearchParams, total int64) {
	w.Header().Set(client.PaginationPageHeader, strconv.Itoa(searchParams.PageNumber))
	w.Header().Set(client.PaginationSizeHeader, strconv.Itoa(searchParams.PageSize))
	w.Header().Set(client.PaginationCountHeader, strconv.FormatInt(total, 10))
}

func getPagination(r *http.Request) (page int, size int) {
	page = 1
	size = defaultPaginationSize

	// Get pagination page
	if param := r.URL.Query()[client.PaginationPageQueryParam]; len(param) == 1 {
		if val, err := strconv.Atoi(param[0]); err == nil {
			page = val
		}
	}

	// Get pagination size
	if param := r.URL.Query()[client.PaginationSizeQueryParam]; len(param) == 1 {
		if val, err := strconv.Atoi(param[0]); err == nil {
			size = val
		}
	}

	// Prevent too much results from being returned
	if size > maxPaginationSize {
		size = maxPaginationSize
	}

	return page, size
}

func getRawQueryParam(url string) map[string]string {
	if url == "" {
		return map[string]string{}
	}

	val := map[string]string{}
	parts := strings.Split(url, "&")

	for _, part := range parts {
		p := strings.Split(part, "=")
		val[p[0]] = p[1]
	}

	return val
}

func writeJSON(w http.ResponseWriter, body interface{}) {
	b, err := json.Marshal(body)
	if err != nil {
		log.Err(err).Msg("error while serializing body")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

func extractResource(msg event.NewResourceEvent) (client.ResourceDto, []string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(msg.Body))
	if err != nil {
		return client.ResourceDto{}, nil, err
	}

	// Get resource title
	title := doc.Find("title").First().Text()

	// Get meta values
	meta := map[string]string{}
	doc.Find("meta").Each(func(i int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		value, _ := s.Attr("content")

		// if name is empty then try to lookup using property
		if name == "" {
			name, _ = s.Attr("property")
			if name == "" {
				return
			}
		}

		meta[strings.ToLower(name)] = value
	})

	// Extract & normalize URLs
	xu := xurls.Strict()
	urls := xu.FindAllString(msg.Body, -1)

	var normalizedURLS []string

	for _, u := range urls {
		normalizedURL, err := normalizeURL(u)
		if err != nil {
			continue
		}

		normalizedURLS = append(normalizedURLS, normalizedURL)
	}

	return client.ResourceDto{
		URL:         msg.URL,
		Body:        msg.Body,
		Time:        msg.Time,
		Title:       title,
		Meta:        meta,
		Description: meta["description"],
	}, normalizedURLS, nil
}

func normalizeURL(u string) (string, error) {
	normalizedURL, err := purell.NormalizeURLString(u, purell.FlagsUsuallySafeGreedy|
		purell.FlagRemoveDirectoryIndex|purell.FlagRemoveFragment|purell.FlagRemoveDuplicateSlashes)
	if err != nil {
		return "", fmt.Errorf("error while normalizing URL %s: %s", u, err)
	}

	return normalizedURL, nil
}
