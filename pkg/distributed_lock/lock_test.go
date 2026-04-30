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
	rdb.Del(ctx, "lock:test:user001") // 清理上次残留

	lock := New(rdb, "lock:test:user001", 5*time.Second)
	ok, err := lock.TryLock(ctx)
	if err != nil {
		t.Fatalf("redis err: %v", err)
	}
	if !ok {
		t.Fatal("first lock should succeed")
	}
}

// 用例 2：同一个 key 第二次加锁应该失败（互斥性）
func TestLock_SecondAcquireFails(t *testing.T) {
	rdb := newTestRedis()
	ctx := context.Background()
	rdb.Del(ctx, "lock:test:user002")

	lockA := New(rdb, "lock:test:user002", 5*time.Second)
	okA, _ := lockA.TryLock(ctx)
	if !okA {
		t.Fatal("lockA should succeed")
	}

	// 模拟另一个进程尝试拿同一把锁
	lockB := New(rdb, "lock:test:user002", 5*time.Second)
	okB, _ := lockB.TryLock(ctx)
	if okB {
		t.Fatal("lockB should fail because lockA still holds it")
	}
}

// 用例 3：TTL 到期后应该可以再次加锁（自动释放）
func TestLock_AutoExpire(t *testing.T) {
	rdb := newTestRedis()
	ctx := context.Background()
	rdb.Del(ctx, "lock:test:user003")

	lockA := New(rdb, "lock:test:user003", 1*time.Second) // 短 TTL 方便测试
	lockA.TryLock(ctx)

	time.Sleep(1500 * time.Millisecond) // 等过期

	lockB := New(rdb, "lock:test:user003", 1*time.Second)
	okB, _ := lockB.TryLock(ctx)
	if !okB {
		t.Fatal("lockB should succeed after lockA expired")
	}
}

// 用例 4：两次 TryLock 成功后写入的 UUID 必须不同（不同 key 才能都成功）
func TestLock_UUIDIsUnique(t *testing.T) {
	rdb := newTestRedis()
	ctx := context.Background()
	rdb.Del(ctx, "lock:test:uuid_a", "lock:test:uuid_b")

	a := New(rdb, "lock:test:uuid_a", 5*time.Second)
	b := New(rdb, "lock:test:uuid_b", 5*time.Second)

	if _, err := a.TryLock(ctx); err != nil {
		t.Fatalf("a TryLock err: %v", err)
	}
	if _, err := b.TryLock(ctx); err != nil {
		t.Fatalf("b TryLock err: %v", err)
	}

	if a.Value() == b.Value() {
		t.Fatal("two lock instances must have different UUIDs")
	}
}
