package service

import (
	"aegis-gateway/internal/global"
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestReserveConcurrency(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	global.Redis = rdb

	ctx := context.Background()

	//读脚本
	content, err := os.ReadFile("../../scripts/lua/reserve.lua")
	if err != nil {
		t.Fatalf("read lua: %v", err)
	}

	sha, err := rdb.ScriptLoad(ctx, string(content)).Result()
	if err != nil {
		t.Fatalf("script load: %v", err)
	}

	global.ReserveSHA = sha

	// 设置初始库存
	// 999代表虚构资源，保证不与真实的userID一样
	rdb.Set(ctx, "resource:stock:999", 1, 0)
	rdb.Del(ctx, "resource:buyers:999")

	//启动200个goroutine。每个调用Reserve()
	var wg sync.WaitGroup
	var successCount atomic.Int32

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func(userID string) {
			defer wg.Done()
			err := Reserve(ctx, userID, 999)
			if err == nil {
				successCount.Add(1)
			}
		}(fmt.Sprintf("user_%d", i))
	}
	wg.Wait() //阻塞，直到计数器归零

	//断言1：只有一个成功
	if got := successCount.Load(); got != 1 {
		t.Errorf("got %d success, want 1", got)
	}

	//断言2：Redis 库存为0
	stock, _ := rdb.Get(ctx, "resource:stock:999").Result()
	if stock != "0" {
		t.Errorf("got stock=%s, want 0", stock)
	}
}
