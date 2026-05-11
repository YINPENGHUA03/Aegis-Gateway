package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"aegis-gateway/internal/bootstrap"
	"aegis-gateway/internal/consumer"
	"aegis-gateway/internal/global"
)

func main() {
	log.Println("Initalizing...")

	//Initailize MYSQL and Redis
	bootstrap.InitMySQL()
	bootstrap.InitRedis()
	bootstrap.InitRabbitMQ()

	consumerCtx, cancelConsumers := context.WithCancel(context.Background())

	// 启动正常订单消费者（常驻 goroutine）
	go consumer.RunOrderConsumer(consumerCtx)
	go consumer.RunDeadLetterConsumer(consumerCtx)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: bootstrap.SetupRouter(),
	}
	go func() {
		log.Println("Aegis-Gateway started, listening port:8080......")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Gateway startup failed: %v", err)
		}
	}()

	//带 1 个缓冲，防止信号在没人接收时丢失
	quit := make(chan os.Signal, 1)
	//告诉 OS：收到 SIGINT（Ctrl+C）或 SIGTERM（docker stop / kill）时，把信号塞进 quit
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server forced to shutdown: %v", err)
	}
	//通知消费者退出循环
	log.Println("HTTP server stopped")
	cancelConsumers()
	log.Println("Consumers stopped")

	if global.MQChannel != nil {
		global.MQChannel.Close()
	}
	log.Println("Shutdown complete")
}
