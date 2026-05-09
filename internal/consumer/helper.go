package consumer

import (
	"log"
	"reflect"
	"strconv"

	amqp "github.com/rabbitmq/amqp091-go"
)

// 提取 Header 中数字的通用安全方法
func getRetryCount(headers amqp.Table, key string) int32 {
	if headers == nil {
		return 0
	}
	val, ok := headers[key]
	if !ok || val == nil {
		return 0
	}

	switch v := val.(type) {
	case int32:
		return v
	case int:
		return int32(v)
	case int8, int16, int64:
		return int32(reflect.ValueOf(v).Int())
	case uint, uint8, uint16, uint32, uint64:
		return int32(reflect.ValueOf(v).Uint())
	case float32:
		return int32(v)
	case float64:
		return int32(v)
	case string:
		n, err := strconv.Atoi(v)
		if err != nil {
			return 0
		}
		return int32(n)
	default:
		log.Printf("[DEAD] unexpected retry-count type: %T", val)
		return 0
	}

}
