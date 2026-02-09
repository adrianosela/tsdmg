// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package client

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/adrianosela/tsdmg/pkg/models"
)

type Client interface {
	CreateRecords(context.Context, *models.CreateRecordsInput) (*models.CreateRecordsOutput, error)
	DeleteRecords(context.Context, *models.DeleteRecordsInput) (*models.DeleteRecordsOutput, error)
}

type client struct {
	httpClient *http.Client
	apiURL     string
}

func New(httpClient *http.Client, apiURL string) Client {
	return &client{
		httpClient: httpClient,
		apiURL:     apiURL,
	}
}

func (c *client) CreateRecords(
	ctx context.Context,
	in *models.CreateRecordsInput,
) (*models.CreateRecordsOutput, error) {
	// Build request.
	var buf bytes.Buffer
	if err := in.Write(&buf); err != nil {
		return nil, fmt.Errorf("failed to marshal csr request: %v", err)
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/dns/records", c.apiURL),
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
		var errResp models.CreateRecordsOutput
		if err := errResp.Read(resp.Body); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("server returned error (status %d): %s", resp.StatusCode, errResp.Error)
		}
		return nil, fmt.Errorf("server returned error (status %d)", resp.StatusCode)
	}

	// Handle success response
	var out models.CreateRecordsOutput
	if err := out.Read(resp.Body); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &out, nil
}

func (c *client) DeleteRecords(
	ctx context.Context,
	in *models.DeleteRecordsInput,
) (*models.DeleteRecordsOutput, error) {
	// Build request.
	var buf bytes.Buffer
	if err := in.Write(&buf); err != nil {
		return nil, fmt.Errorf("failed to marshal csr request: %v", err)
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/dns/records/delete", c.apiURL),
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
		var errResp models.DeleteRecordsOutput
		if err := errResp.Read(resp.Body); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("server returned error (status %d): %s", resp.StatusCode, errResp.Error)
		}
		return nil, fmt.Errorf("server returned error (status %d)", resp.StatusCode)
	}

	// Handle success response
	var out models.DeleteRecordsOutput
	if err := out.Read(resp.Body); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &out, nil
}
