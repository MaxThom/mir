# Security Features

This document describes the security features implemented in the Cockpit server.

## Security Headers

The server automatically adds the following security headers to all responses:

### Content Security Policy (CSP)
```
default-src 'self';
script-src 'self' 'unsafe-inline';
style-src 'self' 'unsafe-inline';
img-src 'self' data: https:;
font-src 'self' data:;
connect-src 'self' ws: wss:;
frame-ancestors 'none'
```

**Purpose:** Prevents XSS attacks by restricting which resources can be loaded.

**Note:** `unsafe-inline` is currently needed for Svelte. In production, consider using a CSP nonce or hash-based approach.

### X-Frame-Options
```
X-Frame-Options: DENY
```

**Purpose:** Prevents clickjacking by denying the page from being embedded in an iframe.

### X-Content-Type-Options
```
X-Content-Type-Options: nosniff
```

**Purpose:** Prevents browsers from MIME-sniffing a response away from the declared content-type.

### X-XSS-Protection
```
X-XSS-Protection: 1; mode=block
```

**Purpose:** Enables the browser's XSS filtering and tells it to block the response if an attack is detected.

### Referrer-Policy
```
Referrer-Policy: strict-origin-when-cross-origin
```

**Purpose:** Controls how much referrer information is included with requests.

### Permissions-Policy
```
Permissions-Policy: geolocation=(), microphone=(), camera=()
```

**Purpose:** Controls which browser features and APIs can be used.

## CORS Configuration

Cross-Origin Resource Sharing (CORS) is configurable to allow requests from specific origins.

### Configuration

In your `cockpit.yaml`:

```yaml
httpServer:
  allowedOrigins:
    - "https://example.com"
    - "http://localhost:5173"  # Svelte dev server
```

### Behavior

- **Empty list**: Allows all origins (development mode) - sets `Access-Control-Allow-Origin: *`
- **Specified origins**: Only allows requests from listed origins
- **Preflight requests**: Automatically handles OPTIONS requests for CORS preflight

### Default Configuration

The default configuration allows:
- `http://localhost:5173` (Svelte dev server)
- `http://localhost:3020` (self)

**⚠️ Production Warning:** Always specify explicit origins in production. Never deploy with an empty `allowedOrigins` list.

## Path Security

### Directory Traversal Protection

The SPA handler includes protection against directory traversal attacks:

```go
// Security: prevent directory traversal
if strings.Contains(requestPath, "..") {
    http.Error(w, "Invalid path", http.StatusBadRequest)
    return
}
```

Any request containing `..` in the path is immediately rejected.

### Path Cleaning

All request paths are cleaned using `path.Clean()` before being processed:

```go
requestPath := path.Clean(r.URL.Path)
```

This normalizes paths and removes redundant elements.

## Cache Control

### Static Assets
```
Cache-Control: public, max-age=31536000
```

Static assets (JS, CSS, images) are cached for 1 year, optimizing performance.

### index.html
```
Cache-Control: no-cache, no-store, must-revalidate
```

The main HTML file is never cached, ensuring users always get the latest version.

## Recommendations for Production

1. **Use HTTPS**: Always deploy with TLS in production
   ```bash
   # Use a reverse proxy like nginx or Caddy for TLS termination
   ```

2. **Specify Allowed Origins**: Never use wildcard CORS in production
   ```yaml
   httpServer:
     allowedOrigins:
       - "https://yourdomain.com"
   ```

3. **Consider CSP Nonce**: For stricter CSP, use nonces instead of `unsafe-inline`

4. **Enable HSTS**: Add Strict-Transport-Security header via reverse proxy
   ```
   Strict-Transport-Security: max-age=31536000; includeSubDomains; preload
   ```

5. **Rate Limiting**: Consider adding rate limiting for API endpoints

6. **Authentication**: Implement proper authentication before deploying to production

## Testing

Run the security tests:

```bash
go test ./cmds/cockpit -v
```

Verify security headers in browser DevTools:
1. Open the application
2. Open DevTools (F12)
3. Go to Network tab
4. Refresh the page
5. Click on the main document
6. Check Response Headers

## Audit

Last security audit: 2026-02-14
Next recommended audit: 2026-05-14

## References

- [OWASP Secure Headers Project](https://owasp.org/www-project-secure-headers/)
- [MDN Web Security](https://developer.mozilla.org/en-US/docs/Web/Security)
- [Content Security Policy Reference](https://content-security-policy.com/)
