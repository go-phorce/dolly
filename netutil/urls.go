package netutil

import (
	"net/url"

	"github.com/juju/errors"
)

// ParseURLs creates a list of URLs from lists of hosts
func ParseURLs(list []string) ([]url.URL, error) {
	urls := make([]url.URL, len(list))
	for i, host := range list {
		u, err := url.Parse(host)
		if err != nil {
			return nil, errors.Trace(err)
		}
		urls[i] = *u
	}

	return urls, nil
}
