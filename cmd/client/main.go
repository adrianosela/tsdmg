// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/adrianosela/tsdmg"
	"github.com/adrianosela/tsdmg/pkg/logger"
	"github.com/adrianosela/tsdmg/pkg/tsautocert"
	"golang.org/x/crypto/acme/autocert"
)

const (
	statusCodeNoError = 0
	statusCodeError   = 1
)

func main() {
	os.Exit(run())
}

func run() int {
	logger := logger.New()
	ctx := context.Background()

	clientOpts := []tsdmg.Option{
		tsdmg.WithLogger(logger),

		// My laptop is already running the Tailscale desktop
		// client, so the tsdmg server is already reachable by
		// node-name i.e. http://tsdmg
		tsdmg.WithSkipTailscaleNode(true),
	}

	client, err := tsdmg.NewClient(ctx, "http://tsdmg", clientOpts...)
	if err != nil {
		logger.Error("failed to initialize client", "error", err)
		return statusCodeError
	}
	defer client.Close()

	logger.Info("tsdmg client initialized")

	records, err := client.Register(ctx)
	if err != nil {
		logger.Error("failed to register node", "error", err)
		return statusCodeError
	}

	logger.Info("node registered with tsdmg server", "records", records)

	certManagerOpts := []tsautocert.Option{
		tsautocert.WithLogger(logger),

		// Cache certificates in the filesystem to
		// avoid hitting the Let's Encrypt rate limit.
		tsautocert.WithCertificateCache(autocert.DirCache("./certcache")),
	}

	certManager, err := tsautocert.NewCertificateManager(ctx, client, "adrianos-macbook.tsdmg.net", certManagerOpts...)
	if err != nil {
		logger.Error("failed to initialize certificate manager", "error", err)
		return statusCodeError
	}
	defer certManager.Close()

	logger.Info("certificate manager initialized")

	if err := certManager.WaitForInitialCert(ctx); err != nil {
		logger.Error("failed to wait for initial certificate", "error", err)
		return statusCodeError
	}
	logger.Info("certificate manager's initial certificate is ready")

	ln, err := net.Listen("tcp", ":443")
	if err != nil {
		logger.Error("failed to start tcp listener on :443", "error", err)
		return statusCodeError
	}

	// Configure TLS listener to get certificate using tsdmg client.
	ln = tls.NewListener(ln, &tls.Config{GetCertificate: certManager.GetCertificate})
	defer ln.Close()

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	// Run server in a goroutine
	errCh := make(chan error, 1)
	defer close(errCh)
	go func() {
		err = http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello World!"))
		}))
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
