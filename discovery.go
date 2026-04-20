package webostv

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/koron/go-ssdp"
)

const (
	SSDP_SERVICE = "urn:schemas-upnp-org:device:MediaRenderer:1"
)

func Discover(ctx context.Context, secure bool) ([]*Client, error) {
	list, err := ssdp.Search(SSDP_SERVICE, 3, "")
	if err != nil {
		return nil, err
	}

	locations := make(map[string]bool)
	for _, res := range list {
		loc := res.Location
		if loc == "" {
			continue
		}
		if locations[loc] {
			continue
		}

		if validateLocation(loc, "LG") {
			locations[loc] = true
		}
	}

	var clients []*Client
	for loc := range locations {
		u, err := url.Parse(loc)
		if err != nil {
			continue
		}
		clients = append(clients, NewClient(u.Hostname(), secure))
	}

	return clients, nil
}

func validateLocation(location, keyword string) bool {
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Get(location)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	return strings.Contains(string(body), keyword)
}
