// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/adrianosela/tsdmg/pkg/dns"
	"github.com/adrianosela/tsdmg/pkg/logger"
	"github.com/adrianosela/tsdmg/pkg/service"
	"github.com/libdns/azure"
	"github.com/libdns/cloudflare"
	"github.com/libdns/godaddy"
	"github.com/libdns/googleclouddns"
	"github.com/libdns/namecheap"
	"github.com/libdns/route53"
	"tailscale.com/tsnet"
)

type stringSlice []string

func (s *stringSlice) String() string         { return strings.Join(*s, ",") }
func (s *stringSlice) Set(value string) error { *s = append(*s, value); return nil }

var (
	allowedProviders = []string{
		"aws",
		"gcp",
		"azure",
		"cloudflare",
		"godaddy",
		"namecheap",
	}
)

var (
	// server
	addr string

	// tailscale config
	tsAuthKey string
	hostname  string

	// dns management config
	domains        stringSlice
	nodeRegDomains stringSlice

	// dns provider creds
	dnsProviderID          string
	awsAccessKeyID         string
	awsSecretAccessKey     string
	awsProfile             string
	gcpProject             string
	gcpServiceAcctJSON     string
	azureSubscriptionID    string
	azureResourceGroupName string
	azureTenantID          string
	azureClientID          string
	azureClientSecret      string
	cloudflareAPIToken     string
	godaddyAPIToken        string
	namecheapAPIKey        string
)

func parseFlags() {
	flag.StringVar(&addr, "addr", ":80", "Address to listen on")
	flag.StringVar(&tsAuthKey, "ts-authkey", "", "Tailscale auth key")
	flag.StringVar(&hostname, "hostname", "tsdmg", "Hostname to use for Tailscale machine")
	flag.Var(&domains, "domain", "Domain management allowlist (repeatable)")
	flag.Var(&nodeRegDomains, "node-reg-domain", "Domain in which to create A/AAAA records for registering Tailscale nodes (repeatable, if \"domain\" is also set, this MUST be a subset of that)")
	flag.StringVar(&dnsProviderID, "dns-provider", "", fmt.Sprintf("Which DNS provider to use (one of [ %s ])", strings.Join(allowedProviders, ", ")))
	flag.StringVar(&awsAccessKeyID, "aws-access-key-id", "", "AWS Access Key ID (used when set if dns-provider is \"aws\")")
	flag.StringVar(&awsSecretAccessKey, "aws-secret-access-key", "", "AWS Secret Access Key (used when set if dns-provider is \"aws\")")
	flag.StringVar(&awsProfile, "aws-profile", "", "AWS Profile (used when set if dns-provider is \"aws\")")
	flag.StringVar(&gcpProject, "gcp-project", "", "Google Cloud Project ID (required if dns-provider is \"gcp\")")
	flag.StringVar(&gcpServiceAcctJSON, "gcp-svc-acct-json", "", "Google Cloud Service Account JSON (used when set if dns-provider is \"gcp\")")
	flag.StringVar(&azureSubscriptionID, "azure-subscription-id", "", "Azure Subscription ID (required if dns-provider is \"azure\")")
	flag.StringVar(&azureResourceGroupName, "azure-resource-group-name", "", "Azure Resource Group Name (required if dns-provider is \"azure\")")
	flag.StringVar(&azureTenantID, "azure-tenant-id", "", "Azure Tenant ID (used when set if dns-provider is \"azure\")")
	flag.StringVar(&azureClientID, "azure-client-id", "", "Azure Client ID (used when set if dns-provider is \"azure\")")
	flag.StringVar(&azureClientSecret, "azure-client-secret", "", "Azure Client Secret (used when set if dns-provider is \"azure\")")
	flag.StringVar(&cloudflareAPIToken, "cloudflare-api-token", "", "Cloudflare API Token (required if dns-provider is \"cloudflare\")")
	flag.StringVar(&godaddyAPIToken, "godaddy-api-token", "", "GoDaddy API Token (required if dns-provider is \"godaddy\")")
	flag.StringVar(&namecheapAPIKey, "namecheap-api-key", "", "Namecheap API Key (required if dns-provider is \"namecheap\")")
	flag.Parse()
}

func getProvider() (dns.Provider, error) {
	switch dnsProviderID {
	case "aws":
		awsProvider := &route53.Provider{
			Region: "us-east-1",
		}
		if awsAccessKeyID != "" {
			awsProvider.AccessKeyId = awsAccessKeyID
		}
		if awsSecretAccessKey != "" {
			awsProvider.SecretAccessKey = awsSecretAccessKey
		}
		if awsProfile != "" {
			awsProvider.Profile = awsProfile
		}
		return awsProvider, nil
	case "gcp":
		if gcpProject == "" {
			return nil, errors.New("flag gcp-project is required but was empty")
		}
		gcpProvider := &googleclouddns.Provider{
			Project: gcpProject,
		}
		if gcpServiceAcctJSON != "" {
			gcpProvider.ServiceAccountJSON = gcpServiceAcctJSON
		}
		return gcpProvider, nil
	case "azure":
		if azureSubscriptionID == "" {
			return nil, errors.New("flag azure-subscription-id is required but was empty")
		}
		if azureResourceGroupName == "" {
			return nil, errors.New("flag azure-resource-group-name is required but was empty")
		}
		azureProvider := &azure.Provider{
			SubscriptionId:    azureSubscriptionID,
			ResourceGroupName: azureResourceGroupName,
		}
		if azureTenantID != "" {
			azureProvider.TenantId = azureTenantID
		}
		if azureClientID != "" {
			azureProvider.ClientId = azureClientID
		}
		if azureClientSecret != "" {
			azureProvider.ClientSecret = azureClientSecret
		}
		return azureProvider, nil
	case "cloudflare":
		if cloudflareAPIToken == "" {
			return nil, errors.New("flag cloudflare-api-token is required but was empty")
		}
		return &cloudflare.Provider{APIToken: cloudflareAPIToken}, nil
	case "godaddy":
		if godaddyAPIToken == "" {
			return nil, errors.New("flag godaddy-api-token is required but was empty")
		}
		return &godaddy.Provider{APIToken: godaddyAPIToken}, nil
	case "namecheap":
		if namecheapAPIKey == "" {
			return nil, errors.New("flag namecheap-api-key is required but was empty")
		}
		return &namecheap.Provider{APIKey: namecheapAPIKey}, nil
	default:
		return nil, fmt.Errorf("invalid dns provider: got %s but must be one of [ %s ]", dnsProviderID, strings.Join(allowedProviders, ", "))
	}
}

const (
	statusCodeNoError = 0
	statusCodeError   = 1
)

func main() {
	os.Exit(run())
}

func run() int {
	logger := logger.New()

	parseFlags()

	if tsAuthKey == "" {
		logger.Error("flag ts-authkey is required but was empty")
		return statusCodeError
	}
	if dnsProviderID == "" {
		logger.Error("flag dns-provider is required but was empty")
		return statusCodeError
	}

	dnsProvider, err := getProvider()
	if err != nil {
		logger.Error("failed to configure dns provider", "error", err)
		return statusCodeError
	}

	srv := new(tsnet.Server)
	srv.AuthKey = tsAuthKey
	srv.Hostname = hostname
	defer func() {
		if err := srv.Close(); err != nil {
			logger.Error("failed to close tsnet server", "error", err)
		}
	}()

	ln, err := srv.Listen("tcp", addr)
	if err != nil {
		logger.Error(
			"failed to initialize TCP listener",
			"addr", addr,
			"error", err,
		)
		return statusCodeError
	}
	defer func() {
		if err := ln.Close(); err != nil {
			logger.Error("failed to close listener", "error", err)
		}
	}()

	tsClient, err := srv.LocalClient()
	if err != nil {
		logger.Error("failed to initialize tailscale local client", "error", err)
		return statusCodeError
	}

	// Wrap tcp listener in tls listener if port is HTTPS port
	if addr == ":443" {
		ln = tls.NewListener(ln, &tls.Config{GetCertificate: tsClient.GetCertificate})
	}

	opts := []service.Option{
		service.WithLogger(logger),
		service.WithDomains(domains...),
		service.WithNodeRegDomains(nodeRegDomains...),
	}
	tsdmg, err := service.New(context.Background(), tsClient, dnsProvider, opts...)
	if err != nil {
		logger.Error("failed to initialize tsdmg service", "error", err)
		return statusCodeError
	}

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	// Run server in a goroutine
	errCh := make(chan error, 1)
	defer close(errCh)
	go func() {
		logger.Info("server starting", "addr", addr)
		err := tsdmg.ServeHTTP(ln)
		errCh <- err
	}()

	// Wait for shutdown signal or error
	select {
	case sig := <-sigCh:
		logger.Info("received signal, shutting down gracefully", "signal", sig.String())
		return statusCodeNoError
	case err := <-errCh:
		if err != nil && !errors.Is(err, net.ErrClosed) {
			logger.Error("server error", "error", err)
			return statusCodeError
		}
		return statusCodeError
	}
}
