package bootstrap

import (
	"aegis-gateway/internal/global"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func InitRabbitMQ() {
	//连接
	conn, err := amqp.Dial("amqp://Dev1atE:yph@666@127.0.0.1:5672/")
	if err != nil {
		log.Fatalf("Connection failed: %v", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Channel failed: %v", err)
	}
	ch.ExchangeDeclare("order_exchange", //交换机名字
		"direct", //类型：direct=精确匹配 routing key 分发
		true,     //durable：持久化 重启之后还在
		false,    //auto-delete:没有Queue绑定的时候自动删除，false：不删除
		false,    //interval:内部交换机互联，false：对外开放
		false,    //no-wait:false=等服务器确认后再继续
		nil)

	ch.QueueDeclare("order_normal_queue",
		true,
		false,
		false,
		false,
		nil)

	delayargs := amqp.Table{
		"x-message-ttl":             int32(15 * 60 * 1000), // TTL 单位是毫秒，15 分钟
		"x-dead-letter-exchange":    "order_exchange",      // 消息死后去哪个交换机
		"x-dead-letter-routing-key": "order_dead",          // 带着哪个暗号去
	}
	ch.QueueDeclare("order_delay_queue",
		true,
		false,
		false,
		false,
		delayargs)

	retryargs := amqp.Table{
		"x-message-ttl":             int32(60 * 1000),
		"x-dead-letter-exchange":    "order_exchange",
		"x-dead-letter-routing-key": "order_normal",
	}
	ch.QueueDeclare("order_retry_queue",
		true,
		false,
		false,
		false,
		retryargs)

	ch.QueueDeclare("order_dead_queue",
		true,
		false,
		false,
		false,
		nil)

	ch.QueueBind("order_normal_queue", // 把哪个 Queue
		"order_normal",   // 用什么 routing key（暗号）
		"order_exchange", // 绑到哪个 Exchange
		false,            // no-wait
		nil)
	ch.QueueBind("order_delay_queue",
		"order_delay",
		"order_exchange",
		false,
		nil)
	ch.QueueBind("order_retry_queue",
		"order_retry",
		"order_exchange",
		false,
		nil)
	ch.QueueBind("order_dead_queue",
		"order_dead",
		"order_exchange",
		false,
		nil)
	returns := ch.NotifyReturn(make(chan amqp.Return, 16))

	go func() {
		for r := range returns {
			log.Printf("MQ message returned! exchange=%s key=%s reason=%s body=%s",
				r.Exchange, r.RoutingKey, r.ReplyText, string(r.Body))
		}
	}()

	global.MQChannel = ch
}
