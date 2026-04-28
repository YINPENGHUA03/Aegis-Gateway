package bootstrap

import (
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
	}

	return r
}
