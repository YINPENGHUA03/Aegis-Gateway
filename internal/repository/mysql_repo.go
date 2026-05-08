package repository

import (
	"context"
	"database/sql"
	"errors"

	"aegis-gateway/internal/global"

	"github.com/google/uuid"
)

// Order maps to a row in the t_order table
type Order struct {
	OrderNo    string
	UserID     string
	ResourceID int64
	Status     int // 0=Pending payment  1=Paid  2=Cancelled
}

// generateOrderNo Generate a globally unique order number based on UUID v4.
// UUID v4 是 122 位随机数，冲突概率 2^-122，在分布式多节点场景下无需协调即可保证唯一。
// 缺点：完全随机会让 InnoDB 主键 B+Tree 频繁页分裂；如果对插入性能敏感，可换 Snowflake。
func generateOrderNo() string {
	return "ORD-" + uuid.NewString()
}

// InsertOrder 向 t_order 表插入一条新订单，返回生成的订单号。
// 被 Day 12 的 RabbitMQ 正常消费者调用：MQ 消息到达 → 调用此函数落库。
func InsertOrder(ctx context.Context, userID string, resourceID int64) (string, error) {
	orderNo := generateOrderNo()

	// 【考点】用 ExecContext 而不是 Exec：
	// ctx 携带超时信息，一旦请求被取消或超时，MySQL 驱动会中断这次查询，
	// 避免慢 SQL 把连接池的连接全部占满（连接泄漏的常见源头）。
	//
	// 【考点】这里不调用 LastInsertId()：
	// 表的主键是 VARCHAR(order_no)，没有自增列，LastInsertId() 会返回 0，没有意义。
	// 我们在插入前就生成了 orderNo，直接返回它即可。
	_, err := global.DB.ExecContext(
		ctx,
		"INSERT INTO t_order (order_no, user_id, resource_id, status) VALUES (?, ?, ?, 0)",
		orderNo,
		userID,
		resourceID,
	)
	if err != nil {
		return "", err
	}

	return orderNo, nil
}

// GetOrderByUserAndResource 按用户+资源查询订单，返回第一条匹配记录。
// 被 Day 13 的死信消费者调用：15 分钟到期 → 调用此函数判断是否已支付。
//
// 返回值约定：
//   - (*Order, nil)  → 找到了记录
//   - (nil, nil)     → 没找到记录（不是错误，只是不存在）
//   - (nil, err)     → 查询本身出错（网络、超时等）
func GetOrderByUserAndResource(ctx context.Context, userID string, resourceID int64) (*Order, error) {
	var o Order

	// 【考点】QueryRow vs Query：
	// Query 返回多行结果集（*sql.Rows），需要手动 rows.Next() 遍历，用完必须 rows.Close()。
	// QueryRow 只取一行，由驱动内部管理资源，不需要手动关闭，代码更简洁。
	// 当你确定结果只有 0 或 1 行时，优先用 QueryRow。
	row := global.DB.QueryRowContext(
		ctx,
		"SELECT order_no, user_id, resource_id, status FROM t_order WHERE user_id = ? AND resource_id = ? LIMIT 1",
		userID,
		resourceID,
	)

	// Scan 将这一行的列值按顺序写入变量，顺序必须和 SELECT 的列顺序完全一致。
	err := row.Scan(&o.OrderNo, &o.UserID, &o.ResourceID, &o.Status)
	if err != nil {
		// 【核心处理】sql.ErrNoRows 是"没有查到记录"的专用哨兵错误，
		// 语义上是正常情况（订单不存在），不应该作为 error 向上抛出。
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &o, nil
}

func UpdateOrderStatus(ctx context.Context, userID string, resourceID int64) (int64, error) {
	result, err := global.DB.ExecContext(
		ctx,
		"UPDATE t_order SET status = 2 WHERE user_id = ? AND resource_id = ? AND status = 0",
		userID,
		resourceID,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
