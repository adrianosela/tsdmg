// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"

	"github.com/adrianosela/tsdmg"
	"github.com/adrianosela/tsdmg/pkg/tsautocert"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"
)

func main() {
	logger := zap.Must(zap.NewProduction())
	ctx := context.Background()

	clientOpts := []tsdmg.Option{
		// Use a real logger.
		tsdmg.WithLogger(logger),

		// My laptop is already running the Tailscale desktop
		// client, so the tsdmg server is already reachable by
		// node-name i.e. http://tsdmg
		tsdmg.WithSkipTailscaleNode(true),
	}

	client, err := tsdmg.NewClient(ctx, "http://tsdmg", clientOpts...)
	if err != nil {
		logger.Fatal("failed to initialize client", zap.Error(err))
	}
	defer client.Close()

	logger.Info("tsdmg client initialized")

	records, err := client.Register(ctx)
	if err != nil {
		logger.Fatal("failed to register node", zap.Error(err))
	}

	logger.Info("node registered with tsdmg server", zap.Any("records", records))

	certManagerOpts := []tsautocert.Option{
		// Use a real logger.
		tsautocert.WithLogger(logger),

		// Cache certificates in the filesystem to
		// avoid hitting the Let's Encrypt rate limit.
		tsautocert.WithCertificateCache(autocert.DirCache("./certcache")),
	}

	certManager, err := tsautocert.NewCertificateManager(ctx, client, "adrianos-macbook.tsdmg.net", certManagerOpts...)
	if err != nil {
		logger.Fatal("failed to initialize certificate manager", zap.Error(err))
	}
	defer certManager.Close()

	logger.Info("certificate manager initialized")

	if err := certManager.WaitForInitialCert(ctx); err != nil {
		logger.Fatal("failed to wait for initial certificate", zap.Error(err))
	}
	logger.Info("certificate manager's initial certificate is ready")

	ln, err := net.Listen("tcp", ":443")
	if err != nil {
		logger.Fatal("failed to start tcp listener on :443", zap.Error(err))
	}

	// Configure TLS listener to get certificate using tsdmg client.
	ln = tls.NewListener(ln, &tls.Config{GetCertificate: certManager.GetCertificate})
	defer ln.Close()

	err = http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World!"))
	}))
	if err != nil {
		logger.Fatal("failed to serve HTTP", zap.Error(err))
	}
}
