package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/kshmirko/lidar-platform-go/internal/utils/auth"
)

const ClaimsKey = "claims"

func AuthMiddleware(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			header := c.Request().Header.Get("Authorization")
			if header == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid authorization format"})
			}

			claims, err := auth.ParseToken(secret, parts[1])
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
			}

			c.Set(ClaimsKey, claims)
			return next(c)
		}
	}
}

func AdminOnly() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			claims := c.Get(ClaimsKey)
			if claims == nil {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "access denied"})
			}

			userClaims, ok := claims.(*auth.Claims)
			if !ok || userClaims.Role != "admin" {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "admin role required"})
			}

			return next(c)
		}
	}
}

// AdminOrManager allows both admin and manager roles.
func AdminOrManager() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			claims := c.Get(ClaimsKey)
			if claims == nil {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "access denied"})
			}

			userClaims, ok := claims.(*auth.Claims)
			if !ok {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "access denied"})
			}

			if userClaims.Role != "admin" && userClaims.Role != "manager" {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "admin or manager role required"})
			}

			return next(c)
		}
	}
}

func GetClaims(c *echo.Context) *auth.Claims {
	claims := c.Get(ClaimsKey)
	if claims == nil {
		return nil
	}
	cv, _ := claims.(*auth.Claims)
	return cv
}
