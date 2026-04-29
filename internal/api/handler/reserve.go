package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Data Transfer Object
type ReserveRequest struct {
	// binding :"required" is the validator engine integrated at the core of Gin
	// 面试防坑：必须对数字类型做边界校验，如 gt=0 (大于0)，防止恶意透传负数或 0 击穿底层逻辑
	UserID     string `json:"user_id" binding:"required,min=5"`
	ResourceID int64  `json:"resource_id" binding:"required,gt=0"`
}

// A unified booking portal exposed to the front end
func HandleReserve(c *gin.Context) {
	var req ReserveRequest

	// 【面试考点】：为什么用 ShouldBindJSON 而不是 BindJSON？
	// BindJSON 在校验失败时会在底层强行写入 HTTP 400 状态码并终止请求，你无法自定义错误返回格式。
	// ShouldBindJSON 把控制权交还给开发者，我们可以封装统一的 JSON 错误协议。
	//gin.H = map[string]interface{} 的简写，用来快速拼一个 key-value 集合给 c.JSON() 序列化输出
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":  400,
			"error": "Invalid request parameters:" + err.Error(),
		})
		return
	}

	// ==========================================
	// TODO (Day 4-10):
	// 1. IP 频率限流
	// 2. 获取 Redis 分布式实名锁 (防重穿透)
	// 3. 执行 Lua 脚本进行内存极速扣减
	// 4. 将扣减成功的消息推入 RabbitMQ
	// ==========================================

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg": `The gateway contract validation has been passed;
		       the request has entered the processing queue.`,
		"data": req,
	})
}
