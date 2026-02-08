package dns01

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/adrianosela/tsdmg/pkg/dns"
	"github.com/libdns/libdns"
	"golang.org/x/crypto/acme"
	"golang.org/x/net/publicsuffix"
)

var (
	ResolverAddrCloudflare = net.UDPAddr{IP: net.ParseIP("1.1.1.1"), Port: 53}
	ResolverAddrGoogle     = net.UDPAddr{IP: net.ParseIP("8.8.8.8"), Port: 53}
	ResolverAddrQuad9      = net.UDPAddr{IP: net.ParseIP("9.9.9.9"), Port: 53}
)

type config struct {
	bundle         bool
	checkResolvers []net.UDPAddr
	checkInterval  time.Duration
}

func (c *config) validate() error {
	if len(c.checkResolvers) == 0 {
		return fmt.Errorf("resolvers list must not be empty")
	}
	if c.checkInterval < time.Second || c.checkInterval > time.Minute*1 {
		return fmt.Errorf("interval must be between 1 and 60 seconds, got %s", c.checkInterval.String())
	}
	return nil
}

type Option func(*config)

func WithBundle(bundle bool) Option {
	return func(c *config) { c.bundle = bundle }
}

func WithPropagationCheckResolvers(resolvers ...net.UDPAddr) Option {
	return func(c *config) { c.checkResolvers = append([]net.UDPAddr{}, resolvers...) }
}

func WithPropagationCheckInterval(interval time.Duration) Option {
	return func(c *config) { c.checkInterval = interval }
}

func GetCertificate(
	ctx context.Context,
	acmeClient *acme.Client,
	dnsProvider dns.Provider,
	csr *x509.CertificateRequest,
	opts ...Option,
) ([][]byte, error) {
	cfg := &config{
		bundle: false,
		checkResolvers: []net.UDPAddr{
			ResolverAddrCloudflare,
			ResolverAddrGoogle,
			ResolverAddrQuad9,
		},
		checkInterval: time.Second * 5,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %v", err)
	}

	if csr.Subject.CommonName == "" {
		return nil, errors.New("CSR does not have a common name (CN)")
	}

	namesForAuthz := append(csr.DNSNames, csr.Subject.CommonName)
	authzIDs := make([]acme.AuthzID, len(namesForAuthz))
	for i, name := range namesForAuthz {
		authzIDs[i] = acme.AuthzID{
			Type:  "dns",
			Value: name,
		}
	}

	// Create ACME order.
	order, err := acmeClient.AuthorizeOrder(ctx, authzIDs)
	if err != nil {
		return nil, err
	}

	// Solve the DNS-01 challenge for all authorizations
	var wg sync.WaitGroup
	errs := make([]error, len(order.AuthzURLs))
	for i, authzURL := range order.AuthzURLs {
		wg.Go(func() {
			authz, err := acmeClient.GetAuthorization(ctx, authzURL)
			if err != nil {
				errs[i] = err
				return
			}
			fqdn := authz.Identifier.Value

			zone, name, err := splitZone(fqdn)
			if err != nil {
				errs[i] = fmt.Errorf("failed to split fqdn: %v", err)
				return
			}

			var chal *acme.Challenge
			for _, c := range authz.Challenges {
				if c.Type == "dns-01" {
					chal = c
					break
				}
			}
			if chal == nil {
				errs[i] = fmt.Errorf("no dns-01 challenge for %s", fqdn)
				return
			}

			txt, err := acmeClient.DNS01ChallengeRecord(chal.Token)
			if err != nil {
				errs[i] = err
				return
			}

			if err := createTXT(ctx, dnsProvider, zone, name, txt); err != nil {
				errs[i] = err
				return
			}

			if err := waitForTXTPropagation(ctx, fqdn, txt, cfg.checkResolvers, cfg.checkInterval); err != nil {
				errs[i] = err
				return
			}
			if _, err := acmeClient.Accept(ctx, chal); err != nil {
				errs[i] = err
				return
			}
			if _, err := acmeClient.WaitAuthorization(ctx, authz.URI); err != nil {
				errs[i] = err
				return
			}
		})
	}
	wg.Wait()

	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("encountered errors while solving dns-01 challenge: %v", err)
	}

	// 4. Finalize order with CSR
	derCerts, _, err := acmeClient.CreateOrderCert(
		ctx,
		order.FinalizeURL,
		csr.Raw,
		cfg.bundle,
	)
	if err != nil {
		return nil, err
	}

	return derCerts, nil
}

func createTXT(
	ctx context.Context,
	dnsProvider dns.Provider,
	zone string,
	name string,
	value string,
) error {

	rec := libdns.TXT{
		Name: "_acme-challenge." + name,
		TTL:  60,
		Text: value,
	}

	_, err := dnsProvider.AppendRecords(ctx, zone, []libdns.Record{rec})
	return err
}

func splitZone(fqdn string) (zone, name string, err error) {
	fqdn = strings.TrimSuffix(fqdn, ".")

	zone, err = publicsuffix.EffectiveTLDPlusOne(fqdn)
	if err != nil {
		return "", "", err
	}

	if fqdn == zone {
		return zone, "", nil
	}

	name = strings.TrimSuffix(fqdn, "."+zone)
	return zone, name, nil
}

func waitForTXTPropagation(
	ctx context.Context,
	fqdn string,
	expected string,
	resolvers []net.UDPAddr,
	interval time.Duration,
) error {

	type result struct {
		ok  bool
		err error
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			var wg sync.WaitGroup
			results := make(chan result, len(resolvers))

			for _, addr := range resolvers {
				wg.Add(1)
				go func(addr net.UDPAddr) {
					defer wg.Done()

					r := resolverFor(addr)
					ok, err := resolverHasTXT(ctx, r, fqdn, expected)
					results <- result{ok: ok, err: err}
				}(addr)
			}

			wg.Wait()
			close(results)

			allOK := true
			for res := range results {
				if res.err != nil || !res.ok {
					allOK = false
					break
				}
			}

			if allOK {
				return nil
			}
		}
	}
}

func resolverFor(addr net.UDPAddr) *net.Resolver {
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, addr.String())
		},
	}
}

func resolverHasTXT(
	ctx context.Context,
	r *net.Resolver,
	fqdn string,
	expected string,
) (bool, error) {

	txts, err := r.LookupTXT(ctx, fqdn)
	if err != nil {
		return false, err
	}

	for _, t := range txts {
		if t == expected {
			return true, nil
		}
	}
	return false, nil
}
