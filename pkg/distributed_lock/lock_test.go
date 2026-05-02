package distributed_lock

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// 测试前置条件：Redis 在 127.0.0.1:6379 运行
func newTestRedis() *redis.Client {
	return redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
}

// 用例 1：第一次加锁应该成功
func TestLock_FirstAcquireSucceeds(t *testing.T) {
	rdb := newTestRedis()
	ctx := context.Background()
	rdb.Del(ctx, "lock:test:user001")

	lock := New(rdb, "lock:test:user001", 5*time.Second)
	ok, err := lock.TryLock(ctx)
	if err != nil {
		t.Fatalf("TryLock() error = %v, want nil", err)
	}
	if !ok {
		t.Fatalf("TryLock() ok = false, want true (clean key, first acquire)")
	}
}

// 用例 2：同一个 key 第二次加锁应该失败（互斥性）
func TestLock_SecondAcquireFails(t *testing.T) {
	rdb := newTestRedis()
	ctx := context.Background()
	rdb.Del(ctx, "lock:test:user002")

	lockA := New(rdb, "lock:test:user002", 5*time.Second)
	okA, err := lockA.TryLock(ctx)
	if err != nil {
		t.Fatalf("lockA.TryLock() error = %v, want nil", err)
	}
	if !okA {
		t.Fatalf("lockA.TryLock() ok = false, want true (clean key)")
	}

	lockB := New(rdb, "lock:test:user002", 5*time.Second)
	okB, err := lockB.TryLock(ctx)
	if err != nil {
		t.Fatalf("lockB.TryLock() error = %v, want nil", err)
	}
	if okB {
		t.Fatalf("lockB.TryLock() ok = true, want false (lockA still holds the key)")
	}
}

// 用例 3：TTL 到期后应该可以再次加锁（自动释放）
func TestLock_AutoExpire(t *testing.T) {
	rdb := newTestRedis()
	ctx := context.Background()
	rdb.Del(ctx, "lock:test:user003")

	lockA := New(rdb, "lock:test:user003", 1*time.Second)
	if _, err := lockA.TryLock(ctx); err != nil {
		t.Fatalf("lockA.TryLock() error = %v, want nil", err)
	}

	time.Sleep(1500 * time.Millisecond)

	lockB := New(rdb, "lock:test:user003", 1*time.Second)
	okB, err := lockB.TryLock(ctx)
	if err != nil {
		t.Fatalf("lockB.TryLock() error = %v, want nil", err)
	}
	if !okB {
		t.Fatalf("lockB.TryLock() ok = false, want true (lockA TTL should have expired after 1.5s)")
	}
}

// 用例 4：两个锁实例的 UUID 必须不同
func TestLock_UUIDIsUnique(t *testing.T) {
	rdb := newTestRedis()
	ctx := context.Background()
	rdb.Del(ctx, "lock:test:uuid_a", "lock:test:uuid_b")

	a := New(rdb, "lock:test:uuid_a", 5*time.Second)
	b := New(rdb, "lock:test:uuid_b", 5*time.Second)

	if _, err := a.TryLock(ctx); err != nil {
		t.Fatalf("a.TryLock() error = %v, want nil", err)
	}
	if _, err := b.TryLock(ctx); err != nil {
		t.Fatalf("b.TryLock() error = %v, want nil", err)
	}

	if a.Value() == b.Value() {
		t.Fatalf("a.Value() == b.Value() == %q, want two distinct UUIDs", a.Value())
	}
}

// 用例 5：加锁后解锁，解锁应该成功，且 Redis key 应该被删除
func TestLock_UnlockSucceeds(t *testing.T) {
	rdb := newTestRedis()
	ctx := context.Background()
	rdb.Del(ctx, "lock:test:user005")

	lock := New(rdb, "lock:test:user005", 5*time.Second)
	if _, err := lock.TryLock(ctx); err != nil {
		t.Fatalf("TryLock() error = %v, want nil", err)
	}

	if !lock.Unlock(ctx) {
		t.Fatalf("Unlock() = false, want true (lock just acquired by this instance)")
	}

	n, err := rdb.Exists(ctx, "lock:test:user005").Result()
	if err != nil {
		t.Fatalf("rdb.Exists() error = %v, want nil", err)
	}
	if n != 0 {
		t.Fatalf("rdb.Exists(key) = %d, want 0 (key should be removed after successful Unlock)", n)
	}
}

// 用例 6：持有错误 UUID 的实例不能解锁别人的锁（Lua 原子比对保护）
func TestLock_WrongUnlock(t *testing.T) {
	rdb := newTestRedis()
	ctx := context.Background()
	rdb.Del(ctx, "lock:test:user006")

	a := New(rdb, "lock:test:user006", 5*time.Second)
	if _, err := a.TryLock(ctx); err != nil {
		t.Fatalf("a.TryLock() error = %v, want nil", err)
	}

	// lockB 持有假 UUID，模拟另一个进程试图释放它不持有的锁
	b := New(rdb, "lock:test:user006", 5*time.Second)
	b.value = "fake-uuid-that-does-not-match"

	if b.Unlock(ctx) {
		t.Fatalf("b.Unlock() = true, want false (Lua should reject mismatched UUID)")
	}

	n, err := rdb.Exists(ctx, "lock:test:user006").Result()
	if err != nil {
		t.Fatalf("rdb.Exists() error = %v, want nil", err)
	}
	if n != 1 {
		t.Fatalf("rdb.Exists(key) = %d, want 1 (lockA's key must not be deleted by failed unlock)", n)
	}
}
