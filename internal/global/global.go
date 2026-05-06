package global

import (
	"database/sql"

	"github.com/redis/go-redis/v9"
)

var (
	DB         *sql.DB       // Global MySQL connection pool instance
	Redis      *redis.Client // Global Redis client instance
	ReserveSHA string        // reserve.lua 上传 Redis 后的 SHA1 指纹，EvalSha 用此替代完整脚本文本
)
