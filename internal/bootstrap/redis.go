package bootstrap

import (
	"context"
	"log"
	"time"

	"aegis-gateway/internal/global"

	"github.com/redis/go-redis/v9"
)

func InitRedis() {
	rdb := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:6379",
		Password:     "",
		DB:           0,
		PoolSize:     200,
		MinIdleConns: 50,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Redis Connection failed: %v", err)
	}

	global.Redis = rdb // Assign to a global variable
	log.Println(" Redis Client initialisation successful!")
}
