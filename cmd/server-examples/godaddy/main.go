// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"crypto/tls"
	"flag"
	"log"

	"github.com/adrianosela/tsdmg/pkg/service"
	"github.com/libdns/godaddy"
	"go.uber.org/zap"
	"tailscale.com/tsnet"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}

	var addr string
	var tsAuthKey string
	var godaddyAPIToken string
	var hostname string

	flag.StringVar(&addr, "addr", ":80", "address to listen on")
	flag.StringVar(&tsAuthKey, "ts-authkey", "", "Tailscale auth key")
	flag.StringVar(&godaddyAPIToken, "godaddy-api-token", "", "GoDaddy API token")
	flag.StringVar(&hostname, "hostname", "tsdmg", "hostname to use for Tailscale machine")
	flag.Parse()

	if tsAuthKey == "" {
		logger.Fatal("flag ts-authkey is required but was empty")
	}
	if godaddyAPIToken == "" {
		logger.Fatal("flag godaddy-api-token is required but was empty")
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

	ctx := context.Background()
	dnsProvider := &godaddy.Provider{APIToken: godaddyAPIToken}

	proxy, err := service.New(ctx, tsClient, dnsProvider, service.WithLogger(logger))
	if err != nil {
		logger.Fatal("failed to initialize tsdmg acme proxy", zap.Error(err))
	}

	if err = proxy.ServeHTTP(ln); err != nil {
		logger.Fatal("failed to serve HTTP over tsnet listener", zap.Error(err))
	}
}
