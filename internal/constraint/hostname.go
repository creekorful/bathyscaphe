package constraint

import (
	configapi "github.com/darkspot-org/bathyscaphe/internal/configapi/client"
	"net/url"
	"strings"
)

// CheckHostnameAllowed check if given URL hostname is allowed
func CheckHostnameAllowed(configClient configapi.Client, rawurl string) (bool, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return false, err
	}

	forbiddenHostnames, err := configClient.GetForbiddenHostnames()
	if err != nil {
		return false, err
	}

	for _, hostname := range forbiddenHostnames {
		if strings.Contains(u.Hostname(), hostname.Hostname) {
			return false, nil
		}
	}

	return true, nil
}
