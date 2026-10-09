# Accounts and local demos

[![AlreadyOn](../web/public/branding/header-doc.svg)](../README.md)

On an empty installation, first-run setup creates the owner. Viewers have restricted permissions and account-specific profiles; the server must enforce those boundaries independently of the interface.

Use synthetic media and disposable storage for public demos. Do not reuse a live owner's credentials or publish a management API key. Development account provisioning is optional and reads ignored local configuration; see [development](DEVELOPMENT.md).

A demo should show guide browsing, tuning, program transitions and profile behavior without exposing filesystem paths or real library metadata. Account and profile limits are part of the API contract in [OpenAPI](../api/openapi.yaml).

Do not reset an existing database to make a demo or test pass. Create a separate temporary instance instead.
