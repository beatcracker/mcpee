# Connecting `mcpee` to ChatGPT

## 1. Create the tunnel

- [OpenAI Platform - Tunnels](https://platform.openai.com/settings/organization/tunnels)

Open **Tunnels** in the OpenAI Platform and click **Create tunnel**. Save the resulting tunnel ID (`tunnel_...`).

## Create a runtime API key

- [OpenAI Platform - API keys](https://platform.openai.com/settings/organization/api-key)

Open **Organization API keys** and create a new secret key.

Choose **Restricted**, then grant:

```text
Tunnels → Read
Tunnels → Use
```

Save the key when it is shown. The tunnel runtime requires both `Read` and `Use` permissions.

## 2. Start the local MCP server

Start `mcpee` and make sure the MCP endpoint is available:

```shell
curl -sS -X POST http://127.0.0.1:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "Mcp-Protocol-Version: 2026-07-28" \
  -H "Mcp-Method: tools/list" \
  --data-binary \
  '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientInfo":{"name":"curl-check","version":"1"},"io.modelcontextprotocol/clientCapabilities":{}}}}'
```

## 3. Start the tunnel client

Keep the API key in an environment variable so it does not end up in shell history or the process command line:

```shell
export CONTROL_PLANE_API_KEY='sk-...'
```

Everything else can be passed explicitly:

```shell
tunnel-client run \
  --control-plane.tunnel-id 'tunnel_...' \
  --control-plane.api-key 'env:CONTROL_PLANE_API_KEY' \
  --mcp.server-url 'http://127.0.0.1:8080/mcp' \
  --mcp.startup-wait-timeout '20s' \
  --health.listen-addr '127.0.0.1:8081' \
  --open-web-ui=false
```

Optionally check tunnel readiness in UI:

```none
http://127.0.0.1:8081/
```

## 4. Create the ChatGPT MCP app

Open **Plugins** in ChatGPT:

- [ChatGPT - Plugins](https://chatgpt.com/plugins?utm_source=chatgpt.com)

Then:

1. Click **Add → Create MCP app**.
2. Select **Tunnel**.
3. Choose the tunnel you created above.
4. Select **No authentication**.
5. Click **Create**.

Keep `tunnel-client` running while creating the app so ChatGPT can discover the MCP tools.

## ⚠️ Refresh tools after MCP changes

If you add, remove, rename, or update MCP servers/tools in `mcpee`:

1. Open **Plugins** in ChatGPT.
2. Find your MCP app.
3. Click **… → Manage**.
4. Scroll down.
5. Click **Refresh tools**.
