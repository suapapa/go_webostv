package webostv

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/koron/go-ssdp"
)

const (
	// SSDPService is the SSDP service type for WebOS TVs.
	SSDPService              = "urn:schemas-upnp-org:device:MediaRenderer:1"
	maxDeviceDescriptionSize = 1 << 20
)

// DiscoverOptions contains options for TV discovery.
type DiscoverOptions struct {
	Secure            bool
	SearchWait        int
	ValidationTimeout time.Duration
}

// DiscoverOption is a functional option for Discover.
type DiscoverOption func(*DiscoverOptions)

// WithDiscoverySecure sets whether discovered clients should use a secure connection.
func WithDiscoverySecure(secure bool) DiscoverOption {
	return func(o *DiscoverOptions) {
		o.Secure = secure
	}
}

// WithSearchWait sets how long to wait for SSDP responses in seconds.
func WithSearchWait(wait int) DiscoverOption {
	return func(o *DiscoverOptions) {
		o.SearchWait = wait
	}
}

// Discover searches for WebOS TVs on the network.
func Discover(ctx context.Context, opts ...DiscoverOption) ([]*Client, error) {
	options := DiscoverOptions{
		Secure:            false,
		SearchWait:        3,
		ValidationTimeout: 5 * time.Second,
	}
	for _, opt := range opts {
		opt(&options)
	}

	list, err := ssdp.Search(SSDPService, options.SearchWait, "")
	if err != nil {
		return nil, fmt.Errorf("ssdp search failed: %w", err)
	}

	locations := map[string]bool{}
	for _, res := range list {
		if res.Location == "" || locations[res.Location] {
			continue
		}

		if validateLocation(
			ctx,
			res.Location,
			"LG",
			options.ValidationTimeout,
		) {
			locations[res.Location] = true
		}
	}

	clients := make([]*Client, 0, len(locations))
	for loc := range locations {
		u, err := url.Parse(loc)
		if err != nil {
			continue
		}
		clients = append(clients, NewClient(u.Hostname(), WithSecure(options.Secure)))
	}

	return clients, nil
}

func validateLocation(ctx context.Context, location, keyword string, timeout time.Duration) bool {
	vCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(
		vCtx,
		http.MethodGet,
		location,
		nil,
	)
	if err != nil {
		return false
	}

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDeviceDescriptionSize))
	if err != nil {
		return false
	}

	return strings.Contains(string(body), keyword)
}
