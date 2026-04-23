package redis

import (
	"context"
	"kama-chat-server/internal/config"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client
var ctx = context.Background()

func init() {
	conf := config.GetConfig()

	RedisClient = redis.NewClient(
		&redis.Options{
			Addr: conf.RedisConfig.Host + ":" + strconv.Itoa(conf.RedisConfig.Port),
			Password: conf.RedisConfig.Password,
			DB: conf.RedisConfig.Db,
		},
	)

	_, err := RedisClient.Ping(ctx).Result()
	if err != nil {
		panic("Redis连接失败:" + err.Error())
	}
}

//  SetKeyEx 设置键值（带过期时间）
func SetKeyEx(key string, value string, timeout time.Duration) error {
	return RedisClient.Set(ctx, key, value, timeout).Err()
}

// GetKey 获取键值（不存在返回空字符串）
func GetKey(key string) (string, error) {
	val, err := RedisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}

	return val, err
}

// DelKeyIfExists 删除单个键
func DelKeyIfExists(key string) error {
	return RedisClient.Del(ctx, key).Err()
}

// DelKeysWithPrefix 删除带前缀的所有键
func DelKeysWithPrefix(prefix string) error {
	keys, err := RedisClient.Keys(ctx, prefix + "*").Result()
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		return RedisClient.Del(ctx, keys...).Err()
	}
	return nil
}



