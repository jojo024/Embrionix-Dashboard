package middleware

import (
	"net/http"
	"strings"

	"github.com/embrionix/dashboard/internal/models"
	"github.com/embrionix/dashboard/internal/services"
	"github.com/gin-gonic/gin"
)

const (
	ctxRole     = "role"
	ctxUsername = "username"
)

// Authenticate validates the request's credentials and stores the caller's role
// and username in the context. When auth is disabled it grants an implicit admin
// so existing deployments keep working unchanged.
func Authenticate(authSvc *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !authSvc.Enabled() {
			c.Set(ctxRole, string(models.RoleAdmin))
			c.Set(ctxUsername, "anonymous")
			c.Next()
			return
		}

		// Static API key (admin-equivalent) for integrations.
		if key := c.GetHeader("X-API-Key"); key != "" && authSvc.APIKeyMatches(key) {
			c.Set(ctxRole, string(models.RoleAdmin))
			c.Set(ctxUsername, "api-key")
			c.Next()
			return
		}

		token := bearerToken(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		username, role, err := authSvc.Verify(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set(ctxRole, string(role))
		c.Set(ctxUsername, username)
		c.Next()
	}
}

// RequireRole rejects callers whose role is below the required level (403).
func RequireRole(required models.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := models.Role(c.GetString(ctxRole))
		if !role.AtLeast(required) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient privileges"})
			return
		}
		c.Next()
	}
}

func bearerToken(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}
