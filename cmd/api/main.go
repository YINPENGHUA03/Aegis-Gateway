package main

import (
	"log"

	"aegis-gateway/internal/bootstrap"
)

func main() {
	log.Println("Initalizing...")

	//Initailize MYSQL and Redis
	bootstrap.InitMySQL()
	bootstrap.InitRedis()
	bootstrap.InitRabbitMQ()
	// Create a default Gin router.
	r := bootstrap.SetupRouter()
	// Log a startup message.
	log.Println("Aegis Gateway started, listening port:8080...")
	// Start the HTTP server on port 8080 and handle potential startup errors.
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Gateway startup failed: %v", err)
	}
}
