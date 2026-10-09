# TLS and remote access

[![AlreadyOn](../web/public/branding/header-doc.svg)](../README.md)

The runtime serves the UI, API and media over HTTPS on port 19900 by default. The HTTP-01 challenge listener uses port 19902 only for certificate validation. Native preview uses its own local HTTPS address.

## Local installations

An instance can provision a local certificate authority for first-run access. Trust that instance's CA only on devices you control. Verify the hostname covered by its certificate. Do not disable certificate validation for an unrelated remote server.

## Public access

Use a domain you control and a publicly trusted certificate. Configure DNS, firewall and reverse-proxy forwarding before requesting certificates. Supported ACME options include HTTP-01 and Cloudflare DNS-01; choose the method appropriate for your deployment.

Keep provider tokens and contact details in private installation settings. Give DNS credentials the minimum required scope. Never commit them to Compose examples or source. Validate renewal and failure reporting before relying on automatic certificates.

## Proxy boundaries

Forward only intended listeners. The application must explicitly trust authorized proxies before accepting forwarded identity or client-address headers. Do not expose administrative endpoints based solely on an unverified forwarded address.

For testing, scoped loopback certificate exceptions are acceptable in tooling; they must not become a product-wide insecure mode. Consult [live testing](AGENT_LIVE_TESTING.md) and [deployment](../deploy/README.md).
