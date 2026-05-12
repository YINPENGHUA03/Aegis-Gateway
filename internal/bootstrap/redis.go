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
		Addr:         os.Getenv("REDIS_ADDR"),
		Password:     os.Getenv("REDIS_PASSWORD"),
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

	content1, err := os.ReadFile("scripts/lua/reserve.lua")
	if err != nil {
		log.Fatalf("Fail to readfile: %v", err)
	}
	script1 := string(content1)

	//Redis用SHA1做脚本缓存的key，节省流量和CPU时间
	sha1, err := rdb.ScriptLoad(ctx, script1).Result()
	if err != nil {
		log.Fatalf("Script load failed: %v", err)
	}
	global.ReserveSHA = sha1

	content2, err := os.ReadFile("scripts/lua/compensate.lua")
	if err != nil {
		log.Fatalf("Fail to readfile: %v", err)
	}
	script2 := string(content2)

	//Redis用SHA1做脚本缓存的key，节省流量和CPU时间
	sha2, err := rdb.ScriptLoad(ctx, script2).Result()
	if err != nil {
		log.Fatalf("Script load failed: %v", err)
	}
	global.CompensateSHA = sha2
}
