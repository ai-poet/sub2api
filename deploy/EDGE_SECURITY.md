# Edge and HTTP Ingress Security

Sub2API supports long-lived SSE and WebSocket requests. Protect the request
ingress without imposing a response `WriteTimeout`: a write deadline would
terminate healthy long generations and streams.

## Application defaults

- `server.max_header_bytes: 65536` limits HTTP/1 request headers to 64 KiB;
  Go maps it to the corresponding HTTP/2 header-list limit.
- `server.read_header_timeout: 10` bounds slow-header attacks. It does not
  limit request processing or response streaming.
- `server.max_request_body_size: 268435456` is the absolute 256 MiB safety net.
- `gateway.max_body_size: 268435456` remains available to multimodal, Gemini,
  image, video, and batch-image endpoints.
- `gateway.text_max_body_size: 33554432` limits the known pure-text
  `/embeddings` and `/alpha/search` endpoints to 32 MiB.
- H2C defaults to 50 concurrent streams per connection, a 2 MiB connection
  upload window, and a 512 KiB stream upload window.
- Invalid credential abuse is limited in process by trusted client IP (IPv6
  `/64`): 120 failures per 60 seconds followed by a 60-second block. This is a
  per-instance safety net; multi-instance enforcement still belongs at the
  load balancer, CDN, or WAF.

Do not add a single application-wide request semaphore: an SSE request may
legitimately occupy it for many minutes. Apply connection and unauthenticated
request controls at the edge; authenticated user/API-key concurrency remains
the application's responsibility.

## Trusted client IPs

`security.trust_forwarded_ip_for_api_key_acl` is enabled by default for upgrade
compatibility. While enabled, raw forwarding headers take over client-IP
resolution for logs and security-sensitive paths. Custom headers from
`security.forwarded_client_ip_headers` are checked in configured order before
the built-in `CF-Connecting-IP`, `X-Real-IP`, and `X-Forwarded-For` fallback.
Header names are case-insensitive, normalized when loaded, de-duplicated, and
limited to 16 unique valid HTTP field names. Header values must contain IP
literals; comma-separated values are supported, invalid entries are skipped,
and public addresses are preferred over private fallback addresses.

The list can be supplied in YAML or with the comma-separated environment
variable `SECURITY_FORWARDED_CLIENT_IP_HEADERS`; an explicitly empty environment
value clears YAML values. It is also editable from the admin security settings
and updates at runtime without a restart. A request snapshots the switch and
header list together, so one request cannot mix old and new settings. Custom
headers are ignored completely when the switch is disabled. In that mode Gin's
`server.trusted_proxies` chain is authoritative: configure only the exact
CIDR/IP addresses that connect directly to Sub2API. An explicit empty list
trusts no forwarded client IPs.

On the first upgrade to this mode, a legacy `false` value is changed to `true`
only when `server.trusted_proxies` was not explicitly configured; explicit
proxy policies remain in secure mode. New installations persist the configured
custom header list during database initialization. Existing installations
backfill a missing database value from the YAML configuration. A hidden
migration marker prevents later administrator changes from being overwritten.
If settings cannot be read or the persisted custom-header list is malformed,
the process fails closed to trusted-proxy mode with no custom headers. If a
migration write fails, the computed mode remains active for the current process
and startup records a warning.

Compatibility takeover accepts forwarded headers without validating the direct
peer, including any configured custom header. Protect the origin from direct
access while it is enabled. A CDN deployment must firewall the origin so only
the CDN or load balancer can reach it, and that proxy must overwrite every
trusted client-IP header rather than append an untrusted client value.

Example for a proxy on the same host:

```yaml
server:
  trusted_proxies:
    - 127.0.0.1/32
    - ::1/128
```

## Nginx baseline

Define shared zones in the `http` block. Tune rates to measured legitimate
traffic; the values below are conservative starting points, not universal
capacity targets.

```nginx
limit_conn_zone $binary_remote_addr zone=sub2api_conn:20m;
limit_req_zone  $binary_remote_addr zone=sub2api_auth:20m rate=5r/s;
limit_req_zone  $binary_remote_addr zone=sub2api_api:40m rate=30r/s;
map $http_upgrade $connection_upgrade {
    default upgrade;
    ''      close;
}

# Credentials in query strings must never reach the access log. The embedded
# pay pages carry the signed-in user's JWT as ?token= (for an admin that is a
# full admin session), and Gemini clients send their API key as ?key=. The
# stock "combined" / "main" formats log both the request line and the Referer,
# and the Referer of every asset a pay page loads repeats its ?token=. When a
# sensitive parameter is present the whole query is replaced.
map $request_uri $sub2api_log_uri {
    default $request_uri;
    "~*^(?<sub2api_log_path>[^?]*)\?(?:[^#]*&)?(?:token|key|api_key|apikey|access_token|refresh_token|secret)=" "${sub2api_log_path}?[REDACTED]";
}
map $http_referer $sub2api_log_referer {
    default $http_referer;
    "~*^(?<sub2api_log_ref>[^?]*)\?(?:[^#]*&)?(?:token|key|api_key|apikey|access_token|refresh_token|secret)=" "${sub2api_log_ref}?[REDACTED]";
}
log_format sub2api_safe '$remote_addr - $remote_user [$time_local] '
                        '"$request_method $sub2api_log_uri $server_protocol" '
                        '$status $body_bytes_sent "$sub2api_log_referer" '
                        '"$http_user_agent" $request_time';

server {
    listen 443 ssl http2;
    server_name api.example.com;

    # Replaces every access_log inherited from the http block for this server.
    access_log /var/log/nginx/sub2api.access.log sub2api_safe;

    client_header_timeout 10s;
    client_max_body_size 256m;
    large_client_header_buffers 4 16k;
    limit_conn sub2api_conn 40;

    # The pay service reaches these routes over loopback (127.0.0.1:8080),
    # never through Nginx. Their token acts as the first admin.
    location ^~ /api/internal/ {
        return 404;
    }

    location ~ ^/(auth|api/auth)/ {
        limit_req zone=sub2api_auth burst=10 nodelay;
        proxy_pass http://127.0.0.1:8080;
    }

    location ~ ^/(v1/)?(embeddings|alpha/search)$ {
        client_max_body_size 32m;
        limit_req zone=sub2api_api burst=60 nodelay;
        proxy_pass http://127.0.0.1:8080;
    }

    location / {
        limit_req zone=sub2api_api burst=60 nodelay;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $remote_addr;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_buffering off;
        proxy_request_buffering off;
        proxy_read_timeout 1800s;
        proxy_send_timeout 1800s;
        proxy_pass http://127.0.0.1:8080;
    }
}
```

Keep credentials out of every Nginx log:

- Any `access_log` set inside a `location` replaces the server-level one, so
  give it the `sub2api_safe` format too. Never add `$request`, `$args`,
  `$query_string`, `$http_referer`, `$http_authorization`, `$http_x_api_key`,
  `$http_x_sub2api_pay_token` or `$http_sec_websocket_protocol` to a format;
  the ops WebSocket carries the admin JWT in that last header.
- `error_log` has no format. When Nginx itself fails a request, for example
  a 502 while the backend restarts or a body over `client_max_body_size`, it
  writes the raw request line and Referer, including `?token=`. Treat the
  error log as sensitive until the pay pages stop putting the token in the URL.
- The `/api/internal/` block assumes the pay service calls the backend over
  loopback. That is the default `SUB2API_INTERNAL_BASE_URL` of the bundled
  image. If a separate pay deployment points it at the public domain, allow
  only that host's address here instead of returning 404.
- Access logs written before this format was in place still hold tokens.
  Delete or restrict them. A logged access token keeps working until it
  expires (`jwt.expire_hour`, 24 h by default); revoking sessions only
  clears refresh tokens, while changing the admin's password invalidates
  every access token issued before it.

If Nginx gzip is enabled in the `http` block, keep `text/event-stream` out of
`gzip_types` and do not use `gzip_types *` for Sub2API. The
`proxy_buffering off` setting above prevents proxy buffering, but it does not
disable the gzip response filter. Use an explicit list for ordinary responses:

```nginx
gzip on;
gzip_types text/plain text/css application/json application/javascript application/xml image/svg+xml;
```

If a shared global configuration cannot exclude SSE by content type, set
`gzip off;` in the locations serving streaming API routes. This leaves gzip
available for the web UI and static assets.

Do not use an incoming `$http_x_forwarded_for` value unless Nginx real-IP
processing is restricted to explicit trusted proxy CIDRs.

## Caddy and CDN

The bundled `deploy/Caddyfile` sets a 64 KiB header limit, a 10-second header
timeout, a 256 MiB absolute body limit, and overwrites forwarded addresses from
the TCP peer. It is therefore a direct-to-Caddy baseline. Do not use its
`{remote_host}` forwarding lines unchanged behind a CDN: all clients would be
attributed to a CDN egress address, collapsing rejection aggregation and the
invalid-auth limiter onto unrelated users.

The bundled Caddy configuration leaves `flush_interval` unset so Caddy can
automatically flush `text/event-stream` responses while still propagating
client cancellation upstream. Do not set it globally: positive values can add
streaming latency, while Caddy 2.6.2's special `-1` mode also causes
reverse-proxied requests to continue after clients disconnect. The
configuration uses an explicit response content-type list for compression. Do
not replace that list with `text/*` or the shorthand `encode gzip zstd`: both
match `text/event-stream` and can buffer SSE until the response ends. Keep
streaming responses uncompressed while retaining compression for the web UI,
JSON, and static assets.

For a CDN deployment, first firewall the origin so only current CDN egress
CIDRs can connect. Then configure those exact ranges as Caddy trusted proxies
and derive upstream headers from Caddy's parsed `{client_ip}`. For example:

```caddyfile
{
	servers {
		trusted_proxies static 192.0.2.0/24 2001:db8:1234::/48
		trusted_proxies_strict
		client_ip_headers CF-Connecting-IP X-Forwarded-For
	}
}

api.example.com {
	reverse_proxy 127.0.0.1:8080 {
		header_up X-Real-IP {client_ip}
		header_up X-Forwarded-For {client_ip}
	}
}
```

Replace the documentation ranges with the CDN's published, automatically
maintained egress ranges. `CF-Connecting-IP` is safe here only because direct
origin access is blocked and Caddy trusts only those TCP peers. Configure
Sub2API `server.trusted_proxies` with the Caddy address/private subnet so the
application accepts only Caddy's rewritten headers.

Caddy core does not provide a general request-rate limiter; use a trusted
CDN/WAF, a supported rate-limit module, or host firewall controls.

At a CDN/WAF, configure connection limits, header/body limits, bot challenges,
and per-IP/ASN rates before traffic reaches the origin. Allow origin ingress
only from CDN egress CIDRs or a private load balancer. Keep the application port
off the public Internet.

## Second CDN domain on the same origin

A deployment may serve the same Sub2API instance on its main domain (for
example `https://api.example.com`) and on an alias domain fronted by another
CDN (for example `https://cdn.example.com`, aimed at users for whom the main
domain is slow). Both domains reach the same process, database and Redis.
Tokens, API keys and desktop login codes are not bound to a domain, so nothing
in the application has to change; the work is in the edge configuration.

**Separate listener for the alias's origin pulls.** Caddy's trusted-proxy
policy is per listener address, not per site, so the alias CDN should pull from
its own port (for example `:8443`), firewalled to that CDN's published egress
ranges. The main listener keeps its own policy:

```caddyfile
{
	servers :443 {
		# direct clients, or the main domain's CDN as described above
	}
	servers :8443 {
		trusted_proxies static 198.51.100.0/24 2001:db8:5678::/48
		trusted_proxies_strict
		client_ip_headers X-Forwarded-For
	}
}

https://cdn.example.com:8443 {
	reverse_proxy 127.0.0.1:8080 {
		header_up X-Real-IP {client_ip}
		header_up X-Forwarded-For {client_ip}
		header_up -CF-Connecting-IP
	}
}
```

Replace the documentation ranges with the CDN's egress ranges, and use the
header in which that CDN reports the visitor's address (many CDNs append it to
`X-Forwarded-For`; some send a vendor-specific header, which then goes first in
`client_ip_headers`). Strip `CF-Connecting-IP` on every listener that is not
behind Cloudflare: in compatibility mode the application reads it before
`X-Real-IP`, so a visitor could otherwise forge it through the alias CDN. Keep
`security.forwarded_client_ip_headers` empty and let Caddy's rewritten
`X-Real-IP` / `X-Forwarded-For` carry the address.

Without this, every visitor of the alias appears as a CDN egress address:
rate limits, the invalid-credential limiter, API-key IP allowlists and session
binding (`session_binding_enabled`) would all act on unrelated users at once.
Check after the switch that the usage log shows real client addresses for
requests made through the alias.

**CDN settings for streaming APIs.**

- First-byte (origin response) timeout of at least 120–300 seconds. Before the
  upstream model answers, the gateway sends nothing except SSE pings while a
  request waits for a concurrency slot (up to 30 seconds), and account
  failover happens in that window too.
- Between-bytes (read) timeout of at least 60 seconds. Streams carry a
  keep-alive every `gateway.stream_keepalive_interval` seconds (default 10)
  once the upstream has answered; keep it enabled.
- Response buffering off, no compression of `text/event-stream`, and no caching
  of anything but static assets. A cached `/v1/models` would also break the
  desktop client's node test, which probes that listing with the user's key.
- Request body limit at least the largest image upload you accept.
- Pass `Authorization`, `x-api-key`, `anthropic-*`, `openai-*` and
  `Idempotency-Key` request headers through unchanged.
- Non-streaming image generation sends nothing until the image is ready; if
  third-party clients will use the alias for it, set
  `gateway.image_nonstream_keepalive_interval: 10`.
- The application sends no WebSocket ping to clients between Codex WebSocket
  turns, so a CDN idle timeout shorter than
  `gateway.openai_ws.ingress_inter_turn_idle_timeout_seconds` can close idle
  sessions. The desktop client configures Codex over HTTP SSE and is not
  affected.

**Web sign-in on the alias.** Third-party OAuth callbacks are registered on the
main domain. Configure the session handoff so OAuth started on the alias
completes there and hands the session back:

```yaml
session_handoff:
  login_origin: "https://api.example.com"
  alias_origins:
    - "https://cdn.example.com"
```

Passkey sign-in on the alias additionally needs the alias in
`webauthn.rp_origins`.

**Desktop client nodes.** The desktop client ships the CDN domain as a built-in
"CDN node" and also offers every bare HTTPS origin listed under Settings → Site
→ Custom endpoints (the list the API Keys page shows), with the name and
description written there. Only gateway traffic follows the node a user picks;
the client's sign-in, account API and payments stay on the main domain.

## DDoS boundary

Application checks reduce amplification after a connection reaches Go. They
cannot absorb volumetric attacks, TLS floods, bandwidth saturation, or a large
distributed source set. Those require upstream network capacity, CDN/WAF
filtering, provider firewall rules, and origin isolation. Avoid high-cardinality
metrics or per-request database security logs during rejection storms.
