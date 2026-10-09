# Server metrics

[![AlreadyOn](../../web/public/branding/header-doc.svg)](../../README.md)

Metrics help owners understand application and host activity. Report CPU, memory, disk and network samples with their timestamps and units. Distinguish application measurements from optional host-wide measurements.

Missing intervals are missing data, not zero utilization. Do not invent complete history after a restart. Network adapter selection and resource totals must describe the actual source rather than silently substituting another interface.

An optional host-metrics helper has a separate deployment and permission boundary. It must not grant broad host access to the application container.

Validate sampling, unavailable sources, partial history and restart behavior with deterministic inputs. Performance conclusions require a measured workload; a chart alone does not prove a bottleneck. See [OpenAPI](../../api/openapi.yaml) for public fields.
