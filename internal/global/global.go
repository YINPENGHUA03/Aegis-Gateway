package global

import (
	"database/sql"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

var (
	DB            *sql.DB       // Global MySQL connection pool instance
	Redis         *redis.Client // Global Redis client instance
	ReserveSHA    string        // reserve.lua 上传 Redis 后的 SHA1 指纹，EvalSha 用此替代完整脚本文本
	CompensateSHA string
	MQChannel     *amqp.Channel // RabbitMQ 通道，用于发布消息到 Exchange

)
