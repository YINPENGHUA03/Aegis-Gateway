package bootstrap

import (
	"aegis-gateway/internal/api/handler"
	"aegis-gateway/internal/api/middleware"
	"aegis-gateway/internal/global"
	"net/http"

	"golang.org/x/time/rate"

	"github.com/gin-gonic/gin"
)

// SetupRouter Configure and return a configured Gin engine
func SetupRouter() *gin.Engine {
	r := gin.Default()

	// 防并发中间件 (Middleware) 全局只挂IP限流
	r.Use(middleware.IPRateLimitMiddleware(rate.Limit(5), 10))

	// API group
	v1 := r.Group("/api/v1")
	{
		v1.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status": "Aegis Gateway Online",
				"mysql":  "connected",
				"redis":  "connected",
			})
		})
		//Registration route
		v1.POST("/reserve", middleware.AntiSpamMiddleware(global.Redis), handler.HandleReserve)
	}

	return r
}
