# mcp-hubspot-go

A [Model Context Protocol](https://modelcontextprotocol.io/) server for HubSpot,
written in Go and shipped as a single static binary.

## Quickstart

```sh
docker run --rm -i \
  -e HUBSPOT_ACCESS_TOKEN=pat-na1-... \
  ghcr.io/zekker6/mcp-hubspot-go:latest
```

For Claude Desktop, add to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "hubspot": {
      "command": "docker",
      "args": ["run", "--rm", "-i",
               "-e", "HUBSPOT_ACCESS_TOKEN",
               "ghcr.io/zekker6/mcp-hubspot-go:latest"],
      "env": {
        "HUBSPOT_ACCESS_TOKEN": "pat-na1-..."
      }
    }
  }
}
```

## Tools

### Read tools (always registered - 12)

| Tool | Inputs |
|---|---|
| `hubspot_get_company` | `company_id`, `properties` (optional string array) |
| `hubspot_get_active_companies` | `limit` (default 10) |
| `hubspot_get_company_activity` | `company_id` |
| `hubspot_get_contact` | `contact_id`, `properties` (optional) |
| `hubspot_get_active_contacts` | `limit` (default 10) |
| `hubspot_get_recent_conversations` | `limit` (default 10), `after` (optional cursor) |
| `hubspot_get_tickets` | `criteria` (`default` \| `Closed`), `limit` (default 50) |
| `hubspot_get_ticket_conversation_threads` | `ticket_id` |
| `hubspot_get_property` | `object_type` (`companies` \| `contacts`), `property_name` |
| `hubspot_get_deal` | `deal_id`, `properties` (optional) |
| `hubspot_get_active_deals` | `limit` (default 10) |
| `hubspot_get_deal_pipelines` | (none) |

### Write tools (gated by `--read-only` - 8)

| Tool | Inputs |
|---|---|
| `hubspot_create_company` | `properties` (object, `name` required) - pre-flight duplicate search by name |
| `hubspot_update_company` | `company_id`, `properties` |
| `hubspot_create_contact` | `properties` (object, `email` required) - pre-flight duplicate search by email |
| `hubspot_update_contact` | `contact_id`, `properties` |
| `hubspot_create_property` | `object_type`, `name`, `label`, `property_type`, `field_type`, `group_name`, `options` (optional) |
| `hubspot_update_property` | `object_type`, `property_name`, plus updatable fields |
| `hubspot_create_deal` | `properties` |
| `hubspot_update_deal` | `deal_id`, `properties` |

`hubspot_create_company` and `hubspot_create_contact` perform an exact-match
search before creating. If a match exists, the tool returns the existing record
with `duplicate: true` and does not create a new one.

## Configuration

| Flag | Env | Default | Purpose |
|---|---|---|---|
| `-access-token` | `HUBSPOT_ACCESS_TOKEN` | _required_ | HubSpot private app token |
| `-read-only` | `HUBSPOT_MCP_READ_ONLY` | `false` | Filter write tools out at registration |
| `-mode` | - | `stdio` | `stdio` \| `sse` \| `http` |
| `-httpListenAddr` | - | `:8012` | Bind address for `sse`/`http` |
| `-httpHeartbeatInterval` | - | `30s` | Keepalive for `http` mode |
| `-sseKeepAliveInterval` | - | `30s` | Keepalive for `sse` mode |
| `-logLevel` | - | `info` | `debug` \| `info` \| `warn` \| `error` |

Flag wins when set explicitly; otherwise the env var fills in. Garbage values
for `HUBSPOT_MCP_READ_ONLY` resolve to `false`.

## Read-only mode

```sh
docker run --rm -i \
  -e HUBSPOT_ACCESS_TOKEN=pat-na1-... \
  ghcr.io/zekker6/mcp-hubspot-go:latest --read-only
```

`tools/list` returns 12 tools (read only). Calling any of the 8 write tool
names returns the framework's "unknown tool" error.

## HTTP and SSE modes

```sh
# Streamable HTTP
docker run --rm -p 8012:8012 \
  -e HUBSPOT_ACCESS_TOKEN=pat-na1-... \
  ghcr.io/zekker6/mcp-hubspot-go:latest \
  -mode=http -httpListenAddr=:8012

# Server-Sent Events
docker run --rm -p 8012:8012 \
  -e HUBSPOT_ACCESS_TOKEN=pat-na1-... \
  ghcr.io/zekker6/mcp-hubspot-go:latest \
  -mode=sse -httpListenAddr=:8012
```

Behind a TLS-terminating proxy (Traefik, nginx, etc.), the in-pod listener is
plain HTTP.

## Build from source

Requires [mise](https://mise.jdx.dev/) and [Task](https://taskfile.dev/).

```sh
mise install
task build       # produces tmp/mcp-hubspot
task test        # unit tests
task e2e         # end-to-end tests (httptest-backed HubSpot fake)
task test:all    # unit + e2e in one shot
task lint        # go vet + gofmt + golangci-lint
task run         # run server in stdio mode (reads HUBSPOT_ACCESS_TOKEN from env)
task run-http    # run server in HTTP mode on :8012
```

`task build` injects the git commit SHA and ISO build date via ldflags; both
appear on the `server starting` log line at startup.

Without mise, install Go 1.26+, Task, and golangci-lint manually, then run the
same `task` targets.

## License

MIT - see [LICENSE](LICENSE).
