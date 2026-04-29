package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// ipBucket 关联一个 IP 的令牌桶 + 它的最近活跃时间
// 最近活跃时间用于定期清理"僵尸 IP"，防止 map 无限增长导致内存泄漏
type ipBucket struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// IPRateLimiter 基于 IP 的限流器集合
type IPRateLimiter struct {
	buckets map[string]*ipBucket
	mu      sync.RWMutex
	rate    rate.Limit // 每秒补 N 个令牌
	burst   int        // 桶容量
}

// NewIPRateLimiter 构造一个限流器集合
// r: 每秒补充令牌数(qps)；b: 桶容量(允许的瞬时突发)
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	limiter := &IPRateLimiter{
		buckets: make(map[string]*ipBucket),
		rate:    r,
		burst:   b,
	}
	go limiter.cleanupLoop() // 启动后台清理协程
	return limiter
}

// getLimiter 获取或创建该 IP 的令牌桶
// 【双重检查锁定模式】先用读锁快速路径，找不到再升级写锁创建
func (i *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	// 快速路径：读锁查找
	i.mu.RLock()
	bucket, exists := i.buckets[ip]
	i.mu.RUnlock()

	if exists {
		bucket.lastSeen = time.Now() // 注意：这里有数据竞争！撕裂读最坏的结果是IP多留5min,但不影响限流正确性
		return bucket.limiter
	}

	// 慢路径：写锁创建
	i.mu.Lock()
	defer i.mu.Unlock()

	// 【DCL 关键】拿到写锁后再次确认（其他 goroutine 可能已经创建）
	bucket, exists = i.buckets[ip]
	if exists {
		bucket.lastSeen = time.Now()
		return bucket.limiter
	}

	// 真正创建
	bucket = &ipBucket{
		limiter:  rate.NewLimiter(i.rate, i.burst),
		lastSeen: time.Now(),
	}
	i.buckets[ip] = bucket
	return bucket.limiter
}

// cleanupLoop 每 5 分钟清理一次超过 10 分钟没活动的 IP
func (i *IPRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		i.mu.Lock()
		for ip, bucket := range i.buckets {
			if time.Since(bucket.lastSeen) > 10*time.Minute {
				delete(i.buckets, ip)
			}
		}
		i.mu.Unlock()
	}
}

// IPRateLimitMiddleware 返回一个 Gin 中间件
// 限流规则：每 IP 每秒 r 个请求，允许 b 个瞬时突发
func IPRateLimitMiddleware(r rate.Limit, b int) gin.HandlerFunc {
	limiter := NewIPRateLimiter(r, b)

	return func(c *gin.Context) {
		ip := c.ClientIP() // Gin 自带的客户端 IP 提取，会处理 X-Forwarded-For
		//安全陷阱：攻击者可以伪造这个头绕过你的限流。

		if !limiter.getLimiter(ip).Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":  429,
				"error": "Too Many Requests: rate limit exceeded",
			})
			c.Abort() // 关键：阻止后续 handler 执行
			return
		}
		c.Next()
	}
}
