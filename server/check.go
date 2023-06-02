package server

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
)

var (
	// Matches valid host names (and ipv4 addresses)
	hostAddressPattern = regexp.MustCompile(`^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])$`)
	errHostParse       = errors.New("Bad host format")
)

func check(host string) (int, error) {
	if !hostAddressPattern.MatchString(host) {
		return -1, errHostParse
	}

	resp, err := http.Get(fmt.Sprintf("http://%s", host))
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}
