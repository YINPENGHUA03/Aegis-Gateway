package consumer

import (
	"aegis-gateway/internal/global"
	"aegis-gateway/internal/repository"
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func retryDeadLetter(msg amqp.Delivery, reason string) {
	//  从包裹的面单（Headers）里提取历史重试次数
	retryCount := getRetryCount(msg.Headers, "x-retry-count")

	//如果已经重试了3次，彻底放弃
	if retryCount >= 3 {
		log.Printf("Failure, Entering manual intervention log, data:%s", msg.Body)
		msg.Ack(false)
		return
	}

	//构造带有新记忆的包裹
	newHeaders := msg.Headers
	if newHeaders == nil {
		newHeaders = make(amqp.Table)
	}
	newHeaders["x-retry-count"] = retryCount + 1

	//重新发货
	err := global.MQChannel.PublishWithContext(
		context.Background(),
		msg.Exchange,
		"order_dead_retry", // 发往延迟轨道，等待 TTL 过期后回流
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
		return
	}
	//销毁过去的旧包裹
	msg.Ack(false)
}

func RunDeadLetterConsumer() {
	deliveries, err := global.MQChannel.Consume(
		"order_dead_queue",
		"",    //consumer 标签，空=自动生成
		false, // autoAck：false=手动确认
		false, // exclusive
		false, // noLocal
		false, // noWait
		nil,
	)
	if err != nil {
		log.Fatalf("Dead letter consume failed: %v", err)
	}

	for msg := range deliveries {
		processOneDeadLetter(msg)
	}
}

func processOneDeadLetter(msg amqp.Delivery) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel() // ← 任何 return 路径都自动执行
	var m OrderMessage
	//1.有毒信息
	if err := json.Unmarshal(msg.Body, &m); err != nil {
		log.Printf("bad message: %v", err)
		msg.Ack(false)
		return
	}

	//2.查订单
	order, err := repository.GetOrderByUserAndResource(ctx, m.UserID, m.ResourceID)
	if err != nil {
		log.Printf("[DEAD] query order failed: %v", err)
		retryDeadLetter(msg, "query failed")
		return
	}

	//3.订单不存在或已支付->Ack
	if order != nil && order.Status == 1 {
		msg.Ack(false)
		return
	}

	//4.未支付->取消订单
	_, err = repository.UpdateOrderStatus(ctx, m.UserID, m.ResourceID)
	if err != nil {
		log.Printf("[DEAD] compensate failed: %v", err)
		retryDeadLetter(msg, "cancel failed")
		return
	}

	//5.补偿Redis(库存+1，移除用户)
	keys := []string{
		"resource:stock:" + strconv.FormatInt(m.ResourceID, 10),
		"resource:buyers:" + strconv.FormatInt(m.ResourceID, 10),
	}
	_, err = global.Redis.EvalSha(ctx, global.CompensateSHA, keys, m.UserID).Result()
	if err != nil {
		log.Printf("[DEAD] compensate failed: %v", err)
		retryDeadLetter(msg, "compensate failed")
		return
	}
	log.Printf("[DEAD] order cancelled: user=%s resource=%d", m.UserID, m.ResourceID)
	msg.Ack(false)

}
