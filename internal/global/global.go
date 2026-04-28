package global

import (
	"database/sql"

	"github.com/redis/go-redis/v9"
)

var (
	DB    *sql.DB       // Global MySQL connection pool instance
	Redis *redis.Client // Global Redis client instance
)
