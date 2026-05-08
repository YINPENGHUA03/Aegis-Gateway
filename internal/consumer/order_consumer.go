package consumer

import (
	"aegis-gateway/internal/global"
	"aegis-gateway/internal/repository"
	"context"
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type OrderMessage struct {
	UserID     string `json:"user_id"`
	ResourceID int64  `json:"resource_id"`
}

func RunOrderConsumer() {
	delivers, err := global.MQChannel.Consume(
		"order_normal_queue",
		"",    //consumer 标签，空=自动生成
		false, // autoAck：false=手动确认
		false, // exclusive
		false, // noLocal
		false, // noWait
		nil,
	)
	if err != nil {
		log.Fatalf("Fail to connect:%v", err)
	}

	for msg := range delivers {
		var m OrderMessage
		//情况一：有毒信息
		if err := json.Unmarshal(msg.Body, &m); err != nil {
			log.Printf("bad message:%v", err)
			msg.Nack(false, false) //丢弃，格式坏
			continue
		}

		_, err := repository.InsertOrder(
			context.Background(),
			m.UserID,
			m.ResourceID)

		//情况二：数据库异常
		if err != nil {
			log.Printf("insert failed : %v", err)

			//  从包裹的面单（Headers）里提取历史重试次数
			var retryCount int32 = 0
			if count, ok := msg.Headers["x-retry-count"].(int32); ok {
				retryCount = count
			}

			//如果已经重试了3次，彻底放弃
			if retryCount >= 3 {
				log.Printf("Failure, Entering manual intervention log, data:%s", msg.Body)
				msg.Ack(false)
				continue
			}

			//构造带有新记忆的包裹
			newHeaders := msg.Headers
			if newHeaders == nil {
				newHeaders = make(amqp.Table)
			}
			newHeaders["x-retry-count"] = retryCount + 1

			//重新发货
			err = global.MQChannel.PublishWithContext(
				context.Background(),
				msg.Exchange,
				"order_delay", // 发往延迟轨道，等待 TTL 过期后回流
				false,
				false,
				amqp.Publishing{
					Headers:      newHeaders,
					ContentType:  msg.ContentType,
					Body:         msg.Body,
					DeliveryMode: amqp.Persistent,
				},
			)

			if err != nil {
				log.Printf("Retry delivery fail: %v", err)
				// 如果连重发 MQ 都失败了，为了不丢数据，只能让原生机制顶上（重回队列）
				msg.Nack(false, true) //数据库出错，重回队列
				continue
			}
			//销毁过去的旧包裹
			msg.Ack(false)
			continue
		}
		//情况三：InsertOrder 第一次就成功
		msg.Ack(false)
	}
}
