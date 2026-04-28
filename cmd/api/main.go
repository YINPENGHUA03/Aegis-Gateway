package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "Aegis Gateway Online",
			"version": "v1.0.0",
		})
	})

	log.Println("Aegis Gateway started, listening port:8080...")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Gateway startup failed: -%v", err)
	}
}
