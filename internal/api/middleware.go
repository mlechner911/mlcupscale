// Copyright (c) 2026 Michael Lechner
// MIT License

package api

import (
    "net/http"
    "strings"
    
    "github.com/gin-gonic/gin"
)

// CORSMiddleware handles Cross-Origin Resource Sharing headers.
func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
    return func(c *gin.Context) {
        origin := c.Request.Header.Get("Origin")
        
        allowed := false
        for _, allowedOrigin := range allowedOrigins {
            if origin == allowedOrigin || allowedOrigin == "*" {
                allowed = true
                break
            }
        }
        
        if allowed {
            c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
            c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
            c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Auth-Token")
        }
        
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }
        
        c.Next()
    }
}

// AuthMiddleware checks for a valid authentication token.
// It supports "Authorization: Bearer <token>" and "X-Auth-Token: <token>".
func AuthMiddleware(validToken string) gin.HandlerFunc {
    return func(c *gin.Context) {
        if validToken == "" {
            c.Next()
            return
        }

        // Check Authorization header
        authHeader := c.GetHeader("Authorization")
        if authHeader != "" {
            parts := strings.Split(authHeader, " ")
            if len(parts) == 2 && parts[0] == "Bearer" && parts[1] == validToken {
                c.Next()
                return
            }
        }

        // Check X-Auth-Token header
        if c.GetHeader("X-Auth-Token") == validToken {
            c.Next()
            return
        }
        
        // Also check query param 'token' for easier browser testing if needed?
        // Let's stick to headers for security best practices.

        c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
    }
}