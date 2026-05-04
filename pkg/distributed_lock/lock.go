package distributed_lock

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// stopOnce 保证 close(stopDogCh) 只执行一次，避免对已关闭 channel 二次 close 导致 panic
type RedisLock struct {
	client    *redis.Client
	key       string
	value     string
	ttl       time.Duration
	mu        sync.Mutex
	stopDogCh chan struct{}
	stopOnce  sync.Once
}

// New 构造一把新锁。仅分配内存，不会真的去 Redis 加锁。
// 调用方需要随后调用 Lock() 才会触发 SETNX。
//
// 参数:
//
//	client: 复用的 Redis 客户端实例（来自 global.Redis）
//	key:    锁的 Redis key，格式 "lock:appoint:{userID}"
//	ttl:    锁的过期时间，
func New(client *redis.Client, key string, ttl time.Duration) *RedisLock {
	return &RedisLock{
		client:    client,
		key:       key,
		ttl:       ttl,
		stopDogCh: make(chan struct{}),
	}
}

func (l *RedisLock) startWatchdog(uuidvalue string) {
	ticker := time.NewTicker(l.ttl / 3)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			//"PEXPIRE" 毫秒精度
			luaScript := `
		if redis.call("GET", KEYS[1])==ARGV[1] then
		    return redis.call("PEXPIRE", KEYS[1], ARGV[2])
		else
		    return 0
		end`
			// ctx 用 context.Background() — 因为业务的 ctx 可能很短，看门狗自己管自己的生命周期
			result, err := l.client.Eval(context.Background(), luaScript, []string{l.key}, uuidvalue, int64(l.ttl/time.Millisecond)).Result()
			if err != nil {
				return
			}
			i, ok := result.(int64)
			if !ok || i == 0 {
				return
			}
		case <-l.stopDogCh:
			return
		}
	}
}

func (l *RedisLock) TryLock(ctx context.Context) (bool, error) {
	value := uuid.New().String()

	// 保证原子性：只有 key 不存在时才设置，并且同时带上过期时间
	result, err := l.client.SetArgs(ctx, l.key, value, redis.SetArgs{
		Mode: "NX",
		TTL:  l.ttl,
	}).Result()

	// 错误处理顺序：先 err 后 result，否则真错误会被当成"锁被占"吞掉
	// 锁被占（NX 失败）：err==redis.Nil，业务正常情况
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	// 真正的故障（网络、Redis 异常等）
	if err != nil {
		return false, err
	}
	// 拿到锁
	if result == "OK" {
		l.mu.Lock()
		l.value = value
		l.stopDogCh = make(chan struct{}) // ← 重新 make
		l.stopOnce = sync.Once{}          // ← Once 也重置
		go l.startWatchdog(value)
		l.mu.Unlock()
		return true, nil
	}
	return false, nil
}

// Value 返回当前锁持有的 UUID（未加锁时为空串）
// Day 6 单元测试和 Lua 解锁脚本调试时会用到
func (l *RedisLock) Value() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.value
}

// LockWithRetry Attempt to acquire the lock repeatedly within the retryTimeout window
func (l *RedisLock) LockWithRetry(ctx context.Context, retryTimeout time.Duration) (bool, error) {
	retryCtx, cancel := context.WithTimeout(ctx, retryTimeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		ok, err := l.TryLock(retryCtx)
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
		case <-ticker.C:
			continue
			// Proceed to the next iteration of the for loop
		}
	}
}

func (l *RedisLock) Unlock(ctx context.Context) bool {
	l.mu.Lock()
	value := l.value
	//对一个已经关闭的 channel 再调一次 close 会 panic。（若业务代码两次调用Unlcok会崩）
	l.stopOnce.Do(func() { close(l.stopDogCh) })
	l.mu.Unlock()

	if value == "" {
		return false
	}

	luaScript := `if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
    else
    return 0
    end`

	result, err := l.client.Eval(ctx, luaScript, []string{l.key}, value).Result()
	if err != nil {
		return false
	}
	i, ok := result.(int64) // 这种写法不会 panic，ok 会告诉你转换是否成功
	if !ok {
		return false
	}
	l.mu.Lock()
	if l.value == value {
		l.value = ""
	}
	l.mu.Unlock()

	return i == 1
}
