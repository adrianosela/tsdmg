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

> My motivation to build this is that I wanted all my internal services and web applications (not on the Internet, accesible only via Tailscale) to be reachable via my custom domain (`<node>.services.adrianosela.com`), and I wanted them to serve HTTPS using public (Let's Encrypt) certificates.
>
> Really I wish I could just delegate the `services.adrianosela.com` zone to Tailscale, and have them do this for me (but it's not possible as of Feb 2026).

## How Does it Work?

Essentially:

- Using Tailscale ACLs, you define which Tailscale sources (nodes, users, groups) can manage which subdomains
- You provision the `tsdmg` service with credentials for your DNS provider (e.g. Cloudflare, Google, GoDaddy, etc...)
- Your Tailscale nodes can request domains to be created/updated/deleted against the `tsdmg` service via HTTP
- The `tsdmg` service will use incoming requests' Tailscale identity to authenticate and authorize (based on Tailscale ACLs) domain management requests

## How Do I Run It?

...TODO: docker