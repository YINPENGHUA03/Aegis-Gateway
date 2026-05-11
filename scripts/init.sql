-- ================================================================
--  Aegis Gateway - Database Schema
--
--  执行方式:
--    docker exec -i appoint_mysql mysql -uroot -p0410 appoint_db < scripts/init.sql
-- ================================================================

SET NAMES utf8mb4;

CREATE TABLE IF NOT EXISTS t_order (
    order_no    VARCHAR(64)  NOT NULL COMMENT '订单号 (UUID v4，ORD- 前缀)',
    user_id     VARCHAR(32)  NOT NULL COMMENT '用户 ID',
    resource_id BIGINT       NOT NULL COMMENT '资源 ID',
    status      TINYINT      NOT NULL DEFAULT 0
                COMMENT '订单状态: 0=待支付  1=已支付  2=已取消',
    created_at  DATETIME     DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME     DEFAULT CURRENT_TIMESTAMP
                                       ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (order_no),
    UNIQUE KEY  uk_user_resource (user_id, resource_id)
                COMMENT '纵深防御：DB 层兜底防重，Redis SISMEMBER 失守时也能拒绝重复订单'
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci
  COMMENT = '预约订单表';

