// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package service

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/adrianosela/tsdmg/pkg/types"
)

type Client interface {
	CreateRecords(context.Context, *types.CreateRecordsInput) (*types.CreateRecordsOutput, error)
	DeleteRecords(context.Context, *types.DeleteRecordsInput) (*types.DeleteRecordsOutput, error)
	Register(context.Context) (*types.RegisterOutput, error)
}

type client struct {
	httpClient *http.Client
	apiURL     string
}

func NewClient(httpClient *http.Client, apiURL string) Client {
	return &client{
		httpClient: httpClient,
		apiURL:     apiURL,
	}
}

func (c *client) CreateRecords(
	ctx context.Context,
	in *types.CreateRecordsInput,
) (*types.CreateRecordsOutput, error) {
	// Build request.
	var buf bytes.Buffer
	if err := in.Write(&buf); err != nil {
		return nil, fmt.Errorf("failed to marshal csr request: %v", err)
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/dns/v1/records", c.apiURL),
		&buf,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build HTTP request object: %v", err)
	}

	// Send request.
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %v", err)
	}

	// Handle error response.
	if resp.StatusCode != http.StatusOK {
		var errResp types.CreateRecordsOutput
		if err := errResp.Read(resp.Body); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("server returned error (status %d): %s", resp.StatusCode, errResp.Error)
		}
		return nil, fmt.Errorf("server returned error (status %d)", resp.StatusCode)
	}

	// Handle success response
	var out types.CreateRecordsOutput
	if err := out.Read(resp.Body); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &out, nil
}

func (c *client) DeleteRecords(
	ctx context.Context,
	in *types.DeleteRecordsInput,
) (*types.DeleteRecordsOutput, error) {
	// Build request.
	var buf bytes.Buffer
	if err := in.Write(&buf); err != nil {
		return nil, fmt.Errorf("failed to marshal csr request: %v", err)
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/dns/v1/records/delete", c.apiURL),
		&buf,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build HTTP request object: %v", err)
	}

	// Send request.
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %v", err)
	}

	// Handle error response.
	if resp.StatusCode != http.StatusOK {
		var errResp types.DeleteRecordsOutput
		if err := errResp.Read(resp.Body); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("server returned error (status %d): %s", resp.StatusCode, errResp.Error)
		}
		return nil, fmt.Errorf("server returned error (status %d)", resp.StatusCode)
	}

	// Handle success response
	var out types.DeleteRecordsOutput
	if err := out.Read(resp.Body); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &out, nil
}

func (c *client) Register(
	ctx context.Context,
) (*types.RegisterOutput, error) {
	// Build request.
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/dns/v1/records/register", c.apiURL),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build HTTP request object: %v", err)
	}

	// Send request.
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %v", err)
	}

	// Handle error response.
	if resp.StatusCode != http.StatusOK {
		var errResp types.RegisterOutput
		if err := errResp.Read(resp.Body); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("server returned error (status %d): %s", resp.StatusCode, errResp.Error)
		}
		return nil, fmt.Errorf("server returned error (status %d)", resp.StatusCode)
	}

	// Handle success response
	var out types.RegisterOutput
	if err := out.Read(resp.Body); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &out, nil
}
