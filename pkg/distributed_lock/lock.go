package distributedlock

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RedisLock struct {
	client    *redis.Client
	key       string
	value     string
	ttl       time.Duration
	mu        sync.Mutex
	stopDogCh chan struct{}
}

// New 构造一把新锁。仅分配内存，不会真的去 Redis 加锁。
// 调用方需要随后调用 Lock() 才会触发 SETNX。
//
// 参数:
//
//	client: 复用的 Redis 客户端实例（来自 global.Redis）
//	key:    锁的 Redis key，建议格式 "lock:appoint:{userID}"
//	ttl:    锁的过期时间，建议 5s（配合 Day 7 看门狗续期）
func New(client *redis.Client, key string, ttl time.Duration) *RedisLock {
	return &RedisLock{
		client:    client,
		key:       key,
		ttl:       ttl,
		stopDogCh: make(chan struct{}),
	}
}

func (l *RedisLock) TryLock(ctx context.Context) (bool, error) {
	//Generate UUID
	value := uuid.New().String()

	result, err := l.client.SetArgs(ctx, l.key, value, redis.SetArgs{
		Mode: "NX",
		TTL:  l.ttl,
	}).Result()

	// Cannot get the lock
	if result != "OK" {
		return false, nil
	}
	if err != nil {
		fmt.Println("Fail to lock")
		return false, err
	}
	if result == "OK" {
		l.mu.Lock()
		l.value = value
		// go l.startWatchdog(value)
		l.mu.Unlock()
		return true, nil
	}
	return false, nil
}

// LockWithRetry Attempt to acquire the lock repeatedly within the retryTimeout window
func (l *RedisLock) LockWithRetry(ctx context.Context, retryTimeout time.Duration) (bool, error) {
	retryCtx, cancel := context.WithTimeout(ctx, retryTimeout)
	defer cancel()

	for {
		ok, err := l.TryLock(ctx)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}

		// The lock is in use; please try again later
		select {
		case <-retryCtx.Done():
			return false, nil
		case <-time.After(500 * time.Millisecond):
			// Proceed to the next iteration of the for loop
		}
	}
}
