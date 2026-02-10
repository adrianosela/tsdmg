// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"crypto/tls"
	"flag"
	"log"
	"strings"

	"github.com/adrianosela/tsdmg/pkg/dns"
	"github.com/adrianosela/tsdmg/pkg/service"
	"github.com/libdns/cloudflare"
	"github.com/libdns/godaddy"
	"go.uber.org/zap"
	"tailscale.com/tsnet"
)

type stringSlice []string

func (s *stringSlice) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}

	var addr string
	var tsAuthKey string
	var dnsProviderID string
	var cloudflareAPIToken string
	var godaddyAPIToken string
	var hostname string
	var regDomains stringSlice
	flag.StringVar(&addr, "addr", ":80", "Address to listen on")
	flag.StringVar(&tsAuthKey, "ts-authkey", "", "Tailscale auth key")
	flag.StringVar(&dnsProviderID, "dns-provider", "", "Which DNS provider to use (cloudflare, godaddy)")
	flag.StringVar(&cloudflareAPIToken, "cloudflare-api-token", "", "Cloudflare API token (required if dns-provider is \"cloudflare\")")
	flag.StringVar(&godaddyAPIToken, "godaddy-api-token", "", "GoDaddy API token (required if dns-provider is \"godaddy\")")
	flag.StringVar(&hostname, "hostname", "tsdmg", "Hostname to use for Tailscale machine")
	flag.Var(&regDomains, "registration-domain", "Domain in which to create A/AAAA records for registering Tailscale nodes (repeatable)")
	flag.Parse()

	if tsAuthKey == "" {
		logger.Fatal("flag ts-authkey is required but was empty")
	}

	var dnsProvider dns.Provider
	switch dnsProviderID {
	case "cloudflare":
		if cloudflareAPIToken == "" {
			logger.Fatal("flag cloudflare-api-token is required but was empty")
		}
		dnsProvider = &cloudflare.Provider{APIToken: cloudflareAPIToken}
	case "godaddy":
		if godaddyAPIToken == "" {
			logger.Fatal("flag godaddy-api-token is required but was empty")
		}
		dnsProvider = &godaddy.Provider{APIToken: godaddyAPIToken}
	}

	srv := new(tsnet.Server)
	srv.AuthKey = tsAuthKey
	srv.Hostname = hostname
	defer func() {
		if err := srv.Close(); err != nil {
			logger.Error("failed to close tsnet server", zap.Error(err))
		}
	}()

	ln, err := srv.Listen("tcp", addr)
	if err != nil {
		logger.Fatal("failed to initialize TCP listener", zap.String("addr", addr), zap.Error(err))
	}
	defer func() {
		if err := ln.Close(); err != nil {
			logger.Error("failed to close listener", zap.Error(err))
		}
	}()

	tsClient, err := srv.LocalClient()
	if err != nil {
		logger.Fatal("failed to initialize tailscale local client", zap.Error(err))
	}

	// wrap tcp listener in tls listener if port is HTTPS port
	if addr == ":443" {
		ln = tls.NewListener(ln, &tls.Config{GetCertificate: tsClient.GetCertificate})
	}

	opts := []service.Option{
		service.WithLogger(logger),
		service.WithRegistration(regDomains...),
	}
	tsdmg, err := service.New(context.Background(), tsClient, dnsProvider, opts...)
	if err != nil {
		logger.Fatal("failed to initialize tsdmg service", zap.Error(err))
	}

	if err = tsdmg.ServeHTTP(ln); err != nil {
		logger.Fatal("failed to serve HTTP over tsnet listener", zap.Error(err))
	}
}
