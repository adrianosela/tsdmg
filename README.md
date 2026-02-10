# Tailscale Domain Mgmt. Gateway (tsdmg)

[![Go Report Card](https://goreportcard.com/badge/github.com/adrianosela/tsdmg)](https://goreportcard.com/report/github.com/adrianosela/tsdmg)
[![Documentation](https://godoc.org/github.com/adrianosela/tsdmg?status.svg)](https://godoc.org/github.com/adrianosela/tsdmg)
[![GitHub issues](https://img.shields.io/github/issues/adrianosela/tsdmg.svg)](https://github.com/adrianosela/tsdmg/issues)
[![license](https://img.shields.io/github/license/adrianosela/tsdmg.svg)](https://github.com/adrianosela/tsdmg/blob/master/LICENSE)

A [tsnet](https://tailscale.com/docs/features/tsnet) based service for managing custom domains in your Tailnet, along with libraries to enable your Tailscale nodes to manage DNS records, and retrieve public (Let's Encrypt) TLS certificates at runtime.

## Why Do I Need This?

Running a `tsdmg` service in your Tailnet enables several use-cases not possible out-of-the-box with Tailscale:

- **Custom domains** for your Tailscale nodes e.g. `<node>.yourdomain.com`
- Allow Tailscale nodes to retrieve **public (Let's Encrypt) TLS certificates** for custom domains
- Allow Tailscale nodes to **manage your domains/subdomains arbitrarily**

> My motivation to build this is that I wanted all my internal services and web applications (not on the Internet, accesible only via Tailscale) to be reachable via my custom domain (`<node>.services.adrianosela.com`), and I wanted them to serve TLS/HTTPS using public (Let's Encrypt) certificates.
>
> Really I wish I could just delegate the `services.adrianosela.com` zone to Tailscale, and have them do this for me (but it's not possible as of Feb 2026).

## How Does it Work?

Essentially:

- Using Tailscale ACLs, you define which Tailscale sources (nodes, users, groups) can manage which subdomains (e.g. node "webapp" can manage "webapp.yourdomain.com")
- You provision the `tsdmg` service with credentials for your DNS provider (e.g. Cloudflare, Google, GoDaddy, etc...)
- Your Tailscale nodes can request domains to be created/updated/deleted against the `tsdmg` service via HTTP
- The `tsdmg` service will use incoming requests' Tailscale identity to authenticate and authorize (based on Tailscale ACLs) domain management requests

## Service Usage

Run the `tsdmg` service as shown in `./cmd/server/main.go`:

```
tsdmg, err := service.New(ctx, tsClient, dnsProvider)
if err != nil {
    logger.Fatal("failed to initialize tsdmg service", zap.Error(err))
}

if err = tsdmg.ServeHTTP(ln); err != nil {
    logger.Fatal("failed to serve HTTP over tsnet listener", zap.Error(err))
}
```

The `dns.Provider` interface is implemented for all major DNS providers by `https://github.com/libdns` e.g.:

- Cloudflare: https://github.com/libdns/cloudflare
- Google Cloud DNS: https://github.com/libdns/googleclouddns
- GoDaddy: https://github.com/libdns/godaddy

## Running the Server with Docker

The `tsdmg` server is available as a Docker image:

```bash
docker run ghcr.io/adrianosela/tsdmg -h
```

### Example: Running with Cloudflare

```bash
docker run -d -it ghcr.io/adrianosela/tsdmg \
  -ts-authkey=$TSDMG_TS_AUTHKEY \
  -dns-provider=cloudflare \
  -cloudflare-api-token=$TSDMG_CLOUDFLARE_API_TOKEN \
  -domain=yourdomain.com \
  -node-reg-domain=yourdomain.com
```

> See the [Makefile](./Makefile) `tsdmg-docker` target for a complete working example.

### Configuration Options

| Flag | Description | Required | Default |
|------|-------------|----------|---------|
| `-addr` | Address to listen on | No | `:80` |
| `-hostname` | Hostname to use for Tailscale machine | No | `tsdmg` |
| `-ts-authkey` | Tailscale auth key | Yes | - |
| `-dns-provider` | Which DNS provider to use (one of `aws`, `gcp`, `azure`, `cloudflare`, `godaddy`) | Yes | - |
| `-domain` | Domain management allowlist (repeatable) | No | - |
| `-node-reg-domain` | Domain in which to create A/AAAA records for registering Tailscale nodes (repeatable, must be subset of `-domain` if both set) | No | - |
| `-cloudflare-api-token` | Cloudflare API Token | If `dns-provider=cloudflare` | - |
| `-aws-access-key-id` | AWS Access Key ID | If `dns-provider=aws` (or use profile) | - |
| `-aws-secret-access-key` | AWS Secret Access Key | If `dns-provider=aws` (or use profile) | - |
| `-aws-profile` | AWS Profile | If `dns-provider=aws` (or use keys) | - |
| `-gcp-project` | Google Cloud Project ID | If `dns-provider=gcp` | - |
| `-gcp-svc-acct-json` | Google Cloud Service Account JSON | If `dns-provider=gcp` | - |
| `-azure-subscription-id` | Azure Subscription ID | If `dns-provider=azure` | - |
| `-azure-resource-group-name` | Azure Resource Group Name | If `dns-provider=azure` | - |
| `-azure-client-id` | Azure Client ID | If `dns-provider=azure` | - |
| `-azure-client-secret` | Azure Client Secret | If `dns-provider=azure` | - |
| `-azure-tenant-id` | Azure Tenant ID | If `dns-provider=azure` | - |
| `-godaddy-api-token` | GoDaddy API Token | If `dns-provider=godaddy` | - |

## Client Usage

This package includes a `tsdmg` client capable of requesting, caching, and refreshing public (Let's Encrypt) TLS certificates, by leveraging a `tsdmg` server.

```
serverURL := "http://tsdmg" // my server's node name is tsdmg

opts := []tsdmg.Option{
	// Use a real logger.
	tsdmg.WithLogger(logger),

	// My laptop is already running the Tailscale desktop
	// client, so the tsdmg server is already reachable by
	// node-name i.e. http://tsdmg. Not setting this option
	// will attempt to initialize a new Tailscale node.
	tsdmg.WithSkipTailscaleNode(true),
}

client, err := tsdmg.NewClient(ctx, serverURL, opts...)
if err != nil {
	logger.Fatal("failed to initialize client", zap.Error(err))
}
defer client.Close()

created, err := client.CreateRecords(ctx, recordsToCreate...)
// check error

deleted, err := client.DeleteRecords(ctx, recordsToDelete...)
// check error
```

## Certificate Manager Usage

With an initialized `tsdmg.Client` client:

> NOTE: import "github.com/adrianosela/tsdmg/pkg/tsautocert"

```
certCommonName := "macbook.tsdmg.net"

opts := []tsautocert.Option{
	// Use a real logger.
	tsautocert.WithLogger(logger),

	// Cache certificates in the filesystem to
	// avoid hitting the Let's Encrypt rate limit.
	tsautocert.WithCertificateCache(autocert.DirCache("./certcache")),
}

certManager, err := tsautocert.NewCertificateManager(
	ctx,
	client,
	certCommonName,
	opts...,
)
if err != nil {
	logger.Fatal("failed to initialize certificate manager", zap.Error(err))
}
defer certManager.Close()

if err := certManager.WaitForInitialCert(ctx); err != nil {
	logger.Fatal("failed to wait for initial certificate", zap.Error(err))
}

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
```


## Tailscale ACLs Example

To allow ANY node to retrieve TLS certificates for `<node>.<your-custom-domain>` (e.g. `your-macbook.yourdomain.com`),
you can add a grant in your ACL as follows:

> The `${node}` will be replaced with the `tsdmg` client node's name by the `tsdmg` server prior to evaluation.

```
	"grants": [
		{
			"src": ["*"],
			"dst": ["*"],
			"ip":  ["*"],

			"app": {
				"tsdmg.net/dns/v1": [
					{
						"TXT": ["_acme-challenge.${node}.yourdomain.com"],
					},
				],
			},
		},
	],
```

Say you also want your client to have the ability to create `A` and `AAAA` records (e.g. for its own tailnet private IP or any other IP):

```
	"grants": [
		{
			"src": ["*"],
			"dst": ["*"],
			"ip":  ["*"],

			"app": {
				"tsdmg.net/dns/v1": [
					{
						"TXT":  ["_acme-challenge.${node}.yourdomain.com"],
						"A":    ["${node}.yourdomain.com"],
						"AAAA": ["${node}.yourdomain.com"],
					},
				],
			},
		},
	],
```

## TODOs:

- Accept generic loggers, not zap.Logger
- Better project structure e.g. `internal`, not everything as `pkg`