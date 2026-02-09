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

type Client interface {
	CreateRecords(context.Context, ...models.Record) ([]models.Record, error)
	DeleteRecords(context.Context, ...models.Record) ([]models.Record, error)

	Close() error
}

type client struct {
	httpClient httpc.Client

	isOpen  atomic.Bool
	closers []func() error
}

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

func (c *client) CreateRecords(
	ctx context.Context,
	records ...models.Record,
) ([]models.Record, error) {
	out, err := c.httpClient.CreateRecords(ctx, &models.CreateRecordsInput{Records: records})
	if err != nil {
		return nil, fmt.Errorf("failed to create records using tsdmg http client: %v", err)
	}
	return out.Records, nil
}

func (c *client) DeleteRecords(
	ctx context.Context,
	records ...models.Record,
) ([]models.Record, error) {
	out, err := c.httpClient.DeleteRecords(ctx, &models.DeleteRecordsInput{Records: records})
	if err != nil {
		return nil, fmt.Errorf("failed to delete records using tsdmg http client: %v", err)
	}
	return out.Records, nil
}

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
