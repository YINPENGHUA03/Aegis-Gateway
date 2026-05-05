package service

import (
	"aegis-gateway/internal/global"
	"context"
	"errors"
	"strconv"
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
		return nil
	}
	return nil
}
