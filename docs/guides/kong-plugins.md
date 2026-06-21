# Kong Custom Plugin Setup Guide

This guide explains how to set up and use custom Kong plugins in the jan-server project.

## Directory Structure

```
integrations/kong/
+-- kong.yml # Main Kong declarative config
+-- kong-dev-full.yml # Dev-Full/Hybrid mode config (host routing)
+-- plugins/ # Custom plugins directory
 +-- keycloak-apikey/ # API key validation plugin
 +-- handler.lua # Plugin logic
 +-- schema.lua # Configuration schema
 +-- README.md # Plugin documentation
```

## Plugin Loading

### Docker Configuration

Kong is configured to load custom plugins via environment variables:

```yaml
environment:
  KONG_PLUGINS: bundled,keycloak-apikey # Load bundled + custom plugins
  KONG_LUA_PACKAGE_PATH: /usr/local/kong/plugins/?.lua;; # Plugin search path

volumes: - ../integrations/kong/plugins:/usr/local/kong/plugins:ro # Mount plugins directory
```

### Verification

After starting Kong, verify plugins are loaded:

```bash
# List all enabled plugins
curl http://localhost:8001/plugins/enabled

# Should include:
# - bundled plugins (jwt, rate-limiting, cors, etc.)
# - keycloak-apikey (custom)
```

## Creating New Plugins

### 1. Create Plugin Directory

```bash
mkdir -p integrations/kong/plugins/my-plugin
```

### 2. Create handler.lua

```lua
local MyPluginHandler = {
 PRIORITY = 1000, -- Plugin execution priority
 VERSION = "1.0.0",
}

function MyPluginHandler:access(conf)
 -- Your plugin logic here
 kong.log.info("My plugin executed!")
end

return MyPluginHandler
```

### 3. Create schema.lua

```lua
return {
 name = "my-plugin",
 fields = {
 { config = {
 type = "record",
 fields = {
 { my_setting = {
 type = "string",
 required = true,
 default = "default_value",
 }},
 }
 }},
 },
}
```

### 4. Register Plugin

Update `infra/docker/infrastructure.yml`:

```yaml
environment:
  KONG_PLUGINS: bundled,keycloak-apikey,my-plugin # Add your plugin
```

### 5. Use in kong.yml

```yaml
plugins:
 - name: my-plugin
 tags: [custom]
 config:
 my_setting: "value"
```

## Plugin Development Tips

### Debugging

1. **Enable debug logging:**

```yaml
environment:
  KONG_LOG_LEVEL: debug
```

2. **Watch logs in real-time:**

```bash
docker logs kong -f
```

3. **Add debug statements:**

```lua
kong.log.debug("Variable value: ", some_variable)
kong.log.err("Error occurred: ", error_message)
```

### Testing Locally

1. **Reload Kong after changes:**

```bash
docker restart kong
```

2. **Test plugin behavior:**

```bash
# Make test request
curl -v http://localhost:8000/your-endpoint \
 -H "X-Custom-Header: value"

# Check response headers
curl -I http://localhost:8000/your-endpoint
```

### Plugin Priority

Kong executes plugins in priority order (higher = earlier):

```
2000+ - Pre-processing (e.g., request transformation)
1000+ - Authentication (e.g., jwt: 1005, keycloak-apikey: 1002)
500+ - Authorization
100+ - Post-processing
```

Set priority in `handler.lua`:

```lua
local MyPluginHandler = {
 PRIORITY = 1002, -- Your priority
}
```

## Common Patterns

### HTTP Requests

```lua
local http = require "resty.http"
local httpc = http.new()

local res, err = httpc:request_uri("http://service:8080/endpoint", {
 method = "POST",
 body = "data",
 headers = {
 ["Content-Type"] = "application/json",
 },
})

if res.status == 200 then
 kong.log.info("Request successful")
end
```

### Header Manipulation

```lua
-- Read headers
local api_key = kong.request.get_header("X-API-Key")

-- Set request headers (to upstream)
kong.service.request.set_header("X-User-ID", "123")

-- Set response headers (to client)
kong.response.set_header("X-Custom", "value")

-- Remove headers
kong.service.request.clear_header("Authorization")
```

### Authentication

```lua
-- Authenticate consumer for rate limiting
kong.client.authenticate({
 id = user_id,
 custom_id = user_subject,
})
```

### Error Responses

```lua
-- Return error to client
return kong.response.exit(401, {
 message = "Unauthorized"
})
```

## Best Practices

1. **Error Handling**: Always handle HTTP errors gracefully
2. **Logging**: Use appropriate log levels (debug, info, warn, err)
3. **Performance**: Cache expensive operations, reuse HTTP connections
4. **Security**: Validate all inputs, sanitize data
5. **Configuration**: Use schema.lua for type-safe config
6. **Testing**: Test with both valid and invalid inputs

## Resources

- [Kong Plugin Development Guide](https://docs.konghq.com/gateway/latest/plugin-development/)
- [Kong PDK Reference](https://docs.konghq.com/gateway/latest/plugin-development/pdk/)
- [Lua Reference](https://www.lua.org/manual/5.1/)

## Troubleshooting

### Plugin Not Loaded

**Symptom**: Plugin not in `/plugins/enabled`

**Solutions**:

1. Check `KONG_PLUGINS` includes your plugin name
2. Verify plugin files are mounted correctly
3. Check file permissions (must be readable)
4. Restart Kong container

### Syntax Errors

**Symptom**: Kong fails to start

**Solutions**:

1. Check Kong logs: `docker logs kong`
2. Validate Lua syntax: `luac -p handler.lua`
3. Check schema format matches Kong requirements

### Plugin Not Executing

**Symptom**: Plugin loaded but not running

**Solutions**:

1. Verify plugin is configured in `kong.yml`
2. Check route/service matches request
3. Ensure priority doesn't conflict with other plugins
4. Add debug logging to verify execution

### Performance Issues

**Symptom**: Slow response times

**Solutions**:

1. Profile plugin execution time
2. Add caching for expensive operations
3. Use connection pooling for HTTP requests
4. Consider async operations if possible
