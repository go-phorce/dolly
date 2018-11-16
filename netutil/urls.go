package netutil

import (
	"net/url"
	"strings"

	"github.com/juju/errors"
)

// ParseURLs creates a list of URLs from lists of hosts
func ParseURLs(list []string) ([]url.URL, error) {
	urls := make([]url.URL, len(list))
	for i, host := range list {
		u, err := url.Parse(strings.TrimSpace(host))
		if err != nil {
			return nil, errors.Trace(err)
		}
		urls[i] = *u
	}

	return urls, nil
}

// ParseURLsFromString creates a list of URLs from a coma-separated lists of hosts
func ParseURLsFromString(hosts string) ([]url.URL, error) {
	hosts = strings.TrimSpace(hosts)
	if len(hosts) == 0 {
		return nil, nil
	}
	list := strings.Split(hosts, ",")
	return ParseURLs(list)
}
