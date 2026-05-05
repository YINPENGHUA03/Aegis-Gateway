package bootstrap

import (
	"context"
	"log"
	"os"
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

	content, err := os.ReadFile("scripts/lua/reserve.lua")
	if err != nil {
		log.Fatalf("Fail to readfile: %v", err)
	}
	script := string(content)

	//Redis用SHA1做脚本缓存的key，节省流量和CPU时间
	sha, err := rdb.ScriptLoad(ctx, script).Result()
	if err != nil {
		log.Fatalf("Script load failed: %v", err)
	}
	global.ReserveSHA = sha
}
