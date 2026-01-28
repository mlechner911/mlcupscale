// Copyright (c) 2026 Michael Lechner
// MIT License

package api

import (
    "github.com/gin-gonic/gin"
)

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
            c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
        }
        
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }
        
        c.Next()
    }
}
