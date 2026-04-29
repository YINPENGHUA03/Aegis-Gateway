package bootstrap

import (
	"aegis-gateway/internal/api/handler"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupRouter Configure and return a configured Gin engine
func SetupRouter() *gin.Engine {
	r := gin.Default()

	// 后续这里可以挂载我们手写的防并发中间件 (Middleware)

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
		v1.POST("/reserve", handler.HandleReserve)
	}

	return r
}
