package app

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// apiContentSecurityPolicy is for JSON responses: nothing in them should ever
// load, run or be framed.
const apiContentSecurityPolicy = "default-src 'none'; frame-ancestors 'none'"

// securityHeaders sets the headers every API response carries. HSTS is only
// sent in production, where the API is always served over HTTPS.
func securityHeaders(hsts bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.Writer.Header()
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		header.Set("X-Frame-Options", "DENY")
		header.Set("Content-Security-Policy", apiContentSecurityPolicy)
		if hsts {
			header.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()
	}
}

// requireAllowedOrigin protects endpoints that act on the refresh cookie from
// cross-site request forgery. A browser sends Origin (or at least Referer) on
// such a POST, so a request from another site is refused even if SameSite
// were ever relaxed. Requests with neither header come from non-browser
// clients, which can't be tricked into sending a victim's cookie, so they pass.
func requireAllowedOrigin(allowedOrigins []string) gin.HandlerFunc {
	allowed := map[string]bool{}
	for _, origin := range allowedOrigins {
		if normalized, ok := normalizeOrigin(origin); ok {
			allowed[normalized] = true
		}
	}
	return func(c *gin.Context) {
		source := c.GetHeader("Origin")
		if source == "" {
			source = c.GetHeader("Referer")
		}
		if source == "" {
			c.Next()
			return
		}
		origin, ok := normalizeOrigin(source)
		if ok && (allowed[origin] || sameOrigin(c.Request, origin)) {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusForbidden, errorResponse("FORBIDDEN", "request origin not allowed"))
	}
}

// normalizeOrigin reduces an Origin or Referer to scheme://host[:port] in
// lower case. "null" and anything without a scheme and host are not origins.
func normalizeOrigin(raw string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", false
	}
	return strings.ToLower(parsed.Scheme + "://" + parsed.Host), true
}

// sameOrigin reports whether the origin's host is the host the request was
// sent to. On Vercel the web app and API share a domain, and the proxy passes
// the public host in Host or X-Forwarded-Host. A cross-site page can't set
// X-Forwarded-Host without a CORS preflight, which it would fail.
func sameOrigin(request *http.Request, origin string) bool {
	host := strings.ToLower(origin[strings.Index(origin, "://")+3:])
	if host == strings.ToLower(request.Host) {
		return true
	}
	forwarded := strings.TrimSpace(strings.Split(request.Header.Get("X-Forwarded-Host"), ",")[0])
	return forwarded != "" && host == strings.ToLower(forwarded)
}
