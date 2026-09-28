# 💦 mcpee

_Because too many MCP routers turned out to be quite poo._

Run multiple local MCP servers (`stdio`), aggregate, expose as HTTP. Done.

🚽 No retry/restart layer, plugin system, policy engine, metrics endpoint, protocol models, authentication or custom JSON-RPC bullshit.

## 📖 Design tenets

1. Catalog projection is configuration.
2. Call routing is a lookup.
3. Everything else belongs to the SDK or outside the process.

## Usage

- [Connecting to ChatGPT](./docs/chatgpt.md)

## Behavior

> [!CAUTION]
> `mcpee` provides no authentication or sandboxing.
>
> Keep it on loopback, or put authentication in front of it if you expose it remotely.
>
> Want sandboxing? Run your MCP binaries through `codex sandbox windows`, `Seatbelt`, `bubblewrap`, or whatever else fits your platform.
>
> `mcpee` doesn't care. 🤗

- frontend: **stateless** Streamable HTTP at `/mcp`
- default listen address: `127.0.0.1:8080`
- backends: `stdio` MCP servers owned for the process lifetime
- projected names: `<backend>__<tool>`
- startup fails if any backend cannot connect/list tools, or if projection is invalid
- descriptions, schemas, annotations, icons, results, errors, and request cancellation are passed through via the official SDK
- logs go ~~brr~~ to stderr

## Configuration

```yaml
listen: 127.0.0.1:8080

backends:
  - name: exec
    command: fs-mcp-rs

  - name: my-stdio-mcp
    command: my-mcp-binary
    args:
      - foxtrot
      - unicorn
    env:
      CHARLIE: KILO
```

## Development

Built on top of `github.com/modelcontextprotocol/go-sdk`. Whatever it needs, we it need too.

Use `mise run` to discover all tasks (`build` / `test` / etc..)

The tests cover catalog projection/routing, metadata preservation, collisions/invalid names, argument forwarding, backend result forwarding, and configuration defaults.
