# Live investigation

[![AlreadyOn](../web/public/branding/header-doc.svg)](../README.md)

A server owner's API key authorizes only the diagnostics they requested. It does not authorize account resets, deployment, filesystem deletion or destructive fixtures.

Keep keys in a user-restricted local file outside Git. Do not print them, embed them in URLs or paste them into issue bodies. Management keys have broad authority; use scoped playback credentials for media requests.

## Procedure

1. Confirm the intended instance and allowed operations.
2. Prefer read-only API and health checks. Record the running revision and relevant configuration without disclosing secrets.
3. Reproduce destructive or full end-to-end scenarios on a synthetic server with temporary storage.
4. Report observations, exact checks and limits. Remove disposable local fixtures after use.

Use HTTPS and validate the certificate for the actual hostname. A trusted local CA is different from a publicly trusted certificate. Keep any loopback-only testing exception narrowly scoped.

Store private instance addresses, keys and operational transcripts under ignored `.private/` or another owner-managed private location. Public issues should contain sanitized reproduction steps and synthetic examples.
