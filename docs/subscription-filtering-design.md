# Subscription filtering and limits design

## Understanding summary

- This fork extends the existing Go CLI so that subscription proxies can be filtered and limited independently for each subscription `tag`.
- A single invocation must continue to support one or many positional `tag=url` subscriptions.
- Name filters support case-insensitive substring matching and case-insensitive glob matching.
- Technical filters support protocol and transport; a country is treated as a convention in the node name rather than being inferred from IP geolocation.
- Include rules are applied first, exclude rules second; values within one attribute are `OR`, while different attributes are `AND`.
- The limit keeps the first `N` matching nodes in the existing deterministic URL-sorted order.
- When no node remains, the generated file contains an empty `outbounds` array.

## Assumptions and constraints

- The first implementation is CLI-first and avoids new dependencies, external geolocation, and a new subscription format.
- Existing invocation and existing global flags retain their current behavior when no new filtering flags are supplied.
- A bare new filter value is global; a qualified value in the form `tag=value` targets one declared subscription tag.
- Global list rules and tag-specific list rules are accumulated. A tag-specific scalar limit overrides a global limit.
- Missing `limit` means unlimited; `limit=0` is valid and produces zero outbounds; negative limits are invalid.
- Repeating a scalar limit for the same scope is an error; repeating list flags appends values.
- Subscriptions are expected to contain tens or hundreds of nodes, so local filtering is sufficient for performance.
- The installed toolchain is Go 1.27.1; `go.mod` remains the compatibility contract at Go 1.25.0 unless implementation evidence requires otherwise.

## Decision log

1. Rules are scoped per `tag` so one invocation can safely process multiple subscriptions with different policies.
2. Repeatable CLI flags are preferred over a config file or an inline subscription mini-language because they require the smallest, dependency-free change and preserve the current command shape.
3. Both global and qualified flag forms are supported: global values are convenient for a single tag or common defaults, while `tag=value` is explicit for multi-tag runs.
4. Name substring and glob matching use separate flags to avoid ambiguous auto-detection and to keep validation and help text clear.
5. Include is evaluated before exclude; matching is deterministic and preserves the existing URL sort order.
6. An empty filtered result is written as an empty `outbounds` list, making generated state reflect the active rules.
7. Country filtering is intentionally name-based; IP geolocation is deferred because it would be less reliable and add data/dependency/network complexity.

## Final design

### CLI contract

New repeatable flags:

```text
--include-name
--include-name-glob
--exclude-name
--exclude-name-glob
--include-protocol
--exclude-protocol
--include-transport
--exclude-transport
--limit
```

Examples:

```bash
# Common rules for one or more tags
xkeen-subscription-watcher home=https://... --include-name Germany --limit 3

# Independent rules in one invocation
xkeen-subscription-watcher \
  home=https://... \
  work=https://... \
  --include-name home=Germany \
  --include-name-glob work=FI-* \
  --include-protocol home=vless \
  --include-transport work=grpc \
  --exclude-name-glob home=*test* \
  --limit home=3 \
  --limit work=5
```

For list flags, a bare value is global and `tag=value` is tag-specific. Values are accumulated. For `--limit`, a bare number is a global default and `tag=number` is a tag-specific override. The left side of a qualified value must match a declared tag; unknown tags are rejected before network requests.

### Internal model and processing

Configuration stores global rules plus a map of per-tag rules. Each rule set contains name substring patterns, name glob patterns, protocol lists, transport lists, and an optional limit. During processing, global and tag-specific list rules are merged and the tag-specific limit takes precedence when present.

`getOutbounds` keeps its existing fetch, sort, parse, tag naming, dialer-proxy, and serialization flow. After parsing each URL it extracts the canonical protocol and transport, obtains the display name with `getProxyName`, evaluates the merged rules, and only then generates outbounds. Parse failures remain per-node warnings and are skipped as today.

Protocol matching is case-insensitive exact matching against `vless`, `vmess`, `trojan`, `ss`, and `hysteria2`. Transport matching is case-insensitive exact matching against the parsed transport (`tcp`, `raw`, `ws`, `grpc`, `xhttp`), plus `hysteria` for Hysteria2. Shadowsocks has no transport value and therefore cannot satisfy a transport include rule. Name substring and glob matching are case-insensitive; invalid glob syntax is a command error.

### Validation and edge cases

- No include rule for an attribute means that attribute imposes no restriction.
- Any matching exclude rule removes the node.
- A node without a name cannot satisfy a name include rule.
- `limit=0` writes an empty `outbounds`; a negative limit fails validation.
- Duplicate scalar limits in the same scope fail instead of silently overwriting.
- A tag with no new rules behaves exactly as before.

### Testing strategy

Add table-driven unit and CLI tests covering name substring/glob include and exclude, protocol and transport matching, OR/AND combinations, global and per-tag rules, limits and zero results, unknown tags, invalid patterns, duplicate limits, preserved URL ordering, unchanged legacy behavior, and generated JSON for multiple subscriptions. Continue using `httptest` fixtures and run `gofmt`, `go vet`, and the race-enabled test command from `AGENTS.md`.

## Risks and deferred scope

- Shell glob expansion requires users to quote patterns containing `*`, `?`, or brackets.
- Name-based country conventions depend on provider naming quality and are not authoritative geolocation.
- A future config file can be added if CLI rule lists become unwieldy; it is intentionally out of scope for the first implementation.

