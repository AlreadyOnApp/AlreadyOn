# Contributing

[![AlreadyOn](web/public/branding/header-doc.svg)](README.md)

Start with [the project brief](docs/PROJECT_BRIEF.md) and [development setup](docs/DEVELOPMENT.md). Discuss substantial changes in an issue before implementation. Assistants must also read [AGENTS.md](AGENTS.md).

## Make a change

1. Start from the maintainer-designated branch. External contributors normally use a fork and a focused branch.
2. Implement one coherent change. Preserve existing behavior outside its scope.
3. Update the relevant guide and [OpenAPI](api/openapi.yaml) when contracts change. Explain unusual logic in source comments.
4. Run relevant checks and describe results in the PR. Include screenshots for visible interface changes and reproduction steps for fixes.

Use `make lint` and `python3 scripts/ci/check-documentation.py` for static checks. Go tests run through `make test`; browser and packaged-runtime tests are described in [delivery](docs/CI_CD.md). Never use live media or a production database as destructive fixtures.

Do not commit secrets, local configuration, signing keys, generated media or test reports. Keep private notes in ignored `.private/`. Ignoring a previously tracked file does not erase its history.

## Review

Use the [PR template](.github/PULL_REQUEST_TEMPLATE.md). Explain the problem, resulting behavior, validation and material risks. Prefer clear boundaries and straightforward code over speculative abstractions. Maintainers decide merge, deployment and release timing.

Report vulnerabilities privately through [SECURITY.md](SECURITY.md). Licensing and third-party redistribution obligations apply to every contribution.
