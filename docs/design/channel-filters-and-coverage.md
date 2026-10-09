# Channel filters and coverage

[![AlreadyOn](../../web/public/branding/header-doc.svg)](../../README.md)

Filters select eligible catalog entries for channel generation. The preview and apply paths must share the same selection semantics so users do not approve one result and receive another.

Combine configured inclusion/exclusion rules explicitly. Metadata availability, unknown ratings, excluded folders and unavailable files must be handled consistently. A broad library channel may include usable items with incomplete metadata; restricted channels require conservative rating rules.

Coverage describes future scheduled programming, not an excuse to rewrite published airings. Refreshes should extend schedules while preserving current entries and rotation state.

Changes need small synthetic examples covering empty selections, overlapping filters, unknown metadata, excluded paths and provider-mode differences. Update [OpenAPI](../../api/openapi.yaml) when request or result schemas change. See [scheduling](../SCHEDULING.md).
