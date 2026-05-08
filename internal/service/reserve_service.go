package service

import (
	"aegis-gateway/internal/global"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	amqp "github.com/rabbitmq/amqp091-go"
)

var ErrSoldOut = errors.New("sold out")
var ErrAlreadyReserved = errors.New("already reserved")

func Reserve(ctx context.Context, userID string, resourceID int64) error {
	// strconv.FormatInt把 int64 类型转换成字符串。
	key1 := "resource:stock:" + strconv.FormatInt(resourceID, 10)
	key2 := "resource:buyers:" + strconv.FormatInt(resourceID, 10)
	keys := []string{key1, key2}
	args := userID

	result, err := global.Redis.EvalSha(ctx, global.ReserveSHA, keys, args).Result()
	i, ok := result.(int64)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("unexpected error")
	}
	switch i {
	case 0:
		return ErrSoldOut
	case -1:
		return ErrAlreadyReserved
	case 1:
		type OrderMessage struct {
			UserID     string `json:"user_id"`
			ResourceID int64  `json:"resource_id"`
		}
		//序列化，把 Go struct 转成 JSON 字节
		body, err := json.Marshal(OrderMessage{UserID: userID, ResourceID: resourceID})
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}
		//order_normal（routing key: order_normal）
		err1 := global.MQChannel.PublishWithContext(ctx,
			"order_exchange",
			"order_normal",
			true, //Exchange 找得到 Queue
			false,
			amqp.Publishing{
				ContentType: "application/json",
				Body:        body,
			},
		)
		if err1 != nil {
			return fmt.Errorf("publish normal: %w", err1)
		}

		//发到 order_delay（routing key: order_delay）
		err2 := global.MQChannel.PublishWithContext(ctx,
			"order_exchange",
			"order_delay",
			true, //Exchange 找得到 Queue
			false,
			amqp.Publishing{
				ContentType: "application/json",
				Body:        body,
			},
		)
		if err2 != nil {
			return fmt.Errorf("publish delay: %w", err2)
		}

		return nil
	}
	return nil
}
