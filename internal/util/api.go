package util

import (
	"fmt"
	"github.com/creekorful/trandoshan/api"
	"github.com/urfave/cli/v2"
	"strings"
)

// GetAPILoginFlag return the cli flag to set api credentials
func GetAPILoginFlag() *cli.StringFlag {
	return &cli.StringFlag{
		Name:     "api-login",
		Usage:    "Login to use when dialing with the API",
		Required: true,
	}
}

// GetAPIURIFlag return the cli flag to set api uri
func GetAPIURIFlag() *cli.StringFlag {
	return &cli.StringFlag{
		Name:     "api-uri",
		Usage:    "URI to the API server",
		Required: true,
	}
}

// GetAPILogin return the credentials from cli flag
func GetAPILogin(c *cli.Context) (api.CredentialsDto, error) {
	if c.String("api-login") == "" {
		return api.CredentialsDto{}, fmt.Errorf("missing credentials")
	}

	credentials := strings.Split(c.String("api-login"), ":")
	if len(credentials) != 2 {
		return api.CredentialsDto{}, fmt.Errorf("wrong credentials format")
	}

	return api.CredentialsDto{Username: credentials[0], Password: credentials[1]}, nil
}

// GetAPIAuthenticatedClient return the authenticated api client
func GetAPIAuthenticatedClient(c *cli.Context) (api.Client, error) {
	// Create the API client
	credentials, err := GetAPILogin(c)
	if err != nil {
		return nil, err
	}
	apiClient, err := api.NewAuthenticatedClient(c.String("api-uri"), credentials)
	if err != nil {
		return nil, err
	}

	return apiClient, nil
}
