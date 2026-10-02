# 💦 mcpee

_Because too many MCP routers turned out to be quite poo._

Run multiple local MCP servers (`stdio`), aggregate, expose as HTTP. Done.

🚽 No request retries, backend supervision, restart policy, plugin system, policy engine, metrics endpoint, protocol models, authentication or custom JSON-RPC bullshit.

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
> Want sandboxing? Run your MCP binaries through [`codex sandbox …`](https://jonny-johnson.medium.com/a-deep-dive-into-codex-windows-sandbox-a2489bf4ae91), `Seatbelt`, `bubblewrap`, or whatever else fits your platform.
>
> `mcpee` doesn't care. 🤗

- frontend: **stateless** Streamable HTTP at `/mcp`
- default listen address: `127.0.0.1:8080`
- backends: `stdio` MCP servers owned for the process lifetime
- backend commands may be wrappers, but must preserve stdin/stdout.
  `mcpee` manages only the configured direct child process
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
    cwd: .
    inherit_env: true
    max_frame_size: 32MiB
```

- `cwd` defaults to mcpee's current working directory, resolved to an absolute path.
- `inherit_env` defaults to `true`. Configured `env` values override inherited values.
- `max_frame_size` accepts bytes or human-readable sizes like `33554432`, `32MB`, `32MiB`, or `-3KiB`.
  The resulting byte count is passed directly to the MCP SDK. `0` keeps the SDK default.
- startup logs report effective runtime values.

## Development

Built on top of [modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk). Whatever it needs, we it need too.

Use `mise run` to discover all tasks (`build` / `test` / etc..)

Tests stick to the what `mcpee` actually owns: projection and routing, pass-through behavior, backend lifecycle, and config semantics.
