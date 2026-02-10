// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package tsdmg

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync/atomic"

	httpc "github.com/adrianosela/tsdmg/pkg/client"
	"github.com/adrianosela/tsdmg/pkg/models"
	"go.uber.org/zap"
	"tailscale.com/client/local"
	"tailscale.com/tsnet"
)

// Client represents a tsdmg client capable of managing DNS records
// by requesting them via the tsdmg server. Aside from arbitrary
// record creation, the client can request to "register" itself
// meaning that the tsdmg server will create A and AAAA records of
// the form ${node}.${domain} on its behalf. The ${domain}s are
// configured on the tsdmg server.
type Client interface {
	// CreateRecords creates the requested DNS records via the tsdmg server.
	CreateRecords(context.Context, ...models.Record) ([]models.Record, error)

	// DeleteRecords deletes the requested DNS records via the tsdmg server.
	DeleteRecords(context.Context, ...models.Record) ([]models.Record, error)

	// Register requests the tsdmg server to create A and AAAA records for this
	// node's Tailscale private IPs. Domain configuration lives server side. That
	// is, the tsdmg server decides which domains to create records in, but all
	// records will be of the form ${node}.${domain}.
	Register(context.Context) ([]models.Record, error)

	// Close closes the client gracefully.
	Close() error
}

type client struct {
	httpClient httpc.Client

	isOpen  atomic.Bool
	closers []func() error
}

// NewClient returns a new Client for a tsdmg server at serverURL with
// the given options. Note that serverURL must include scheme (http/s).
func NewClient(ctx context.Context, serverURL string, opts ...Option) (Client, error) {
	cfg := &config{
		logger:            zap.NewNop(),
		serverURL:         serverURL,
		skipTailscaleNode: false,
		tailscaleClient:   nil,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Slice for functions to be called on Client's Close().
	// NOTE: they will be closed in reverse order e.g. LIFO.
	var closers []func() error

	// Initialize tailscale client if none provided via options.
	if cfg.tailscaleClient == nil && !cfg.skipTailscaleNode {
		srv := new(tsnet.Server)
		srv.Ephemeral = true
		if err := srv.Start(); err != nil {
			return nil, fmt.Errorf("failed to start Tailscale tsnet node: %v", err)
		}
		closers = append(closers, srv.Close)

		tsClient, err := srv.LocalClient()
		if err != nil {
			if closeErr := srv.Close(); closeErr != nil {
				cfg.logger.Error("failed to close Tailscale tsnet node")
			}
			return nil, fmt.Errorf("failed to initialize tailscale local client: %v", err)
		}
		cfg.tailscaleClient = tsClient
	}

	httpClient := http.DefaultClient
	if cfg.tailscaleClient != nil {
		httpClient = httpClientFromTsClient(cfg.tailscaleClient)
	}

	return &client{
		httpClient: httpc.New(httpClient, cfg.serverURL),
		isOpen:     atomic.Bool{},
		closers:    closers,
	}, nil
}

// CreateRecords creates the requested DNS records via the tsdmg server.
func (c *client) CreateRecords(ctx context.Context, records ...models.Record) ([]models.Record, error) {
	out, err := c.httpClient.CreateRecords(ctx, &models.CreateRecordsInput{Records: records})
	if err != nil {
		return nil, fmt.Errorf("failed to create records using tsdmg http client: %v", err)
	}
	return out.Records, nil
}

// DeleteRecords deletes the requested DNS records via the tsdmg server.
func (c *client) DeleteRecords(ctx context.Context, records ...models.Record) ([]models.Record, error) {
	out, err := c.httpClient.DeleteRecords(ctx, &models.DeleteRecordsInput{Records: records})
	if err != nil {
		return nil, fmt.Errorf("failed to delete records using tsdmg http client: %v", err)
	}
	return out.Records, nil
}

// Register requests the tsdmg server to create A and AAAA records for this
// node's Tailscale private IPs. Domain configuration lives server side. That
// is, the tsdmg server decides which domains to create records in, but all
// records will be of the form ${node}.${domain}.
func (c *client) Register(ctx context.Context) ([]models.Record, error) {
	out, err := c.httpClient.Register(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to register node using tsdmg http client: %v", err)
	}
	return out.Records, nil
}

// Close closes the client gracefully.
func (c *client) Close() error {
	if !c.isOpen.CompareAndSwap(true, false) {
		return errClientClosed
	}

	// Run closers in reverse order.
	errs := []error{}
	for i := len(c.closers) - 1; i >= 0; i-- {
		errs = append(errs, c.closers[i]())
	}

	return errors.Join(errs...)
}

// httpClientFromTsClient returns an httpClient for which all
// requests will go over the given Tailscale local client.
func httpClientFromTsClient(tsClient *local.Client) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return tsClient.Dial(ctx, network, addr)
			},
		},
		// NOTE: request timeouts will be context-controller, no need to define any here...
	}
}
