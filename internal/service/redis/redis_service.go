package redis

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"kama-chat-server/internal/config"
	"kama-chat-server/pkg/zlog"
	"log"
	"strconv"
	"time"
)

var redisClient *redis.Client
var ctx = context.Background()

func init() {
	conf := config.GetConfig()
	host := conf.RedisConfig.Host
	port := conf.RedisConfig.Port
	password := conf.RedisConfig.Password
	db := conf.Db
	addr := host + ":" + strconv.Itoa(port)

	redisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}

func SetKeyEx(key string, value string, timeout time.Duration) error {
	err := redisClient.Set(ctx, key, value, timeout).Err()
	if err != nil {
		return err
	}
	return nil
}

func GetKey(key string) (string, error) {
	value, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			zlog.Info("该key不存在")
			return "", nil
		}
		return "", err
	}
	return value, nil
}

func GetKeyNilIsErr(key string) (string, error) {
	value, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return value, nil
}

// GetKeyWithPrefixNilIsErr 根据前缀查找单个key（不存在返回redis.Nil错误）
func GetKeyWithPrefixNilIsErr(prefix string) (string, error) {
	var cursor uint64
	var foundKeys []string

	for {
		// 使用Scan增量迭代，每次返回100条，不阻塞Redis
		keys, nextCursor, err := redisClient.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			return "", err
		}

		// 收集找到的键
		foundKeys = append(foundKeys, keys...)

		// 更新cursor继续迭代
		cursor = nextCursor
		// cursor为0表示迭代完成
		if cursor == 0 {
			break
		}
	}

	if len(foundKeys) == 0 {
		zlog.Info("没有找到相关前缀key")
		return "", redis.Nil
	}

	if len(foundKeys) == 1 {
		zlog.Info(fmt.Sprintln("成功找到了相关前缀key", foundKeys))
		return foundKeys[0], nil
	} else {
		zlog.Error("找到了数量大于1的key，查找异常")
		return "", errors.New("找到了数量大于1的key，查找异常")
	}
}

// GetKeyWithSuffixNilIsErr 根据后缀查找单个key（不存在返回redis.Nil错误）
func GetKeyWithSuffixNilIsErr(suffix string) (string, error) {
	var cursor uint64
	var foundKeys []string

	for {
		// 使用Scan增量迭代，每次返回100条，不阻塞Redis
		keys, nextCursor, err := redisClient.Scan(ctx, cursor, "*"+suffix, 100).Result()
		if err != nil {
			return "", err
		}

		// 收集找到的键
		foundKeys = append(foundKeys, keys...)

		// 更新cursor继续迭代
		cursor = nextCursor
		// cursor为0表示迭代完成
		if cursor == 0 {
			break
		}
	}

	if len(foundKeys) == 0 {
		zlog.Info("没有找到相关后缀key")
		return "", redis.Nil
	}

	if len(foundKeys) == 1 {
		zlog.Info(fmt.Sprintln("成功找到了相关后缀key", foundKeys))
		return foundKeys[0], nil
	} else {
		zlog.Error("找到了数量大于1的key，查找异常")
		return "", errors.New("找到了数量大于1的key，查找异常")
	}
}

func DelKeyIfExists(key string) error {
	exists, err := redisClient.Exists(ctx, key).Result()
	if err != nil {
		return err
	}
	if exists == 1 { // 键存在
		delErr := redisClient.Del(ctx, key).Err()
		if delErr != nil {
			return delErr
		}
	}
	// 无论键是否存在，都不返回错误
	return nil
}

// DelKeysWithPattern 删除精确匹配pattern的key
// ★使用Scan增量迭代，不会阻塞Redis，适合生产环境
func DelKeysWithPattern(pattern string) error {
	var cursor uint64
	for {
		// 使用Scan代替Keys，每次返回100条，不阻塞Redis
		keys, nextCursor, err := redisClient.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}

		// 删除找到的键
		if len(keys) > 0 {
			if err := redisClient.Del(ctx, keys...).Err(); err != nil {
				return err
			}
			log.Println("成功删除相关对应key", keys)
		}

		// 更新cursor继续迭代
		cursor = nextCursor
		// cursor为0表示迭代完成
		if cursor == 0 {
			break
		}
	}

	return nil
}

func DelKeysWithPrefix(prefix string) error {
	// 使用Scan增量迭代，不会阻塞Redis，适合生产环境
	var cursor uint64
	for {
		keys, nextCursor, err := redisClient.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			return err
		}

		// 删除找到的键
		if len(keys) > 0 {
			if err := redisClient.Del(ctx, keys...).Err(); err != nil {
				return err
			}
			log.Println("成功删除相关前缀key", keys)
		}

		// 更新cursor继续迭代
		cursor = nextCursor
		// cursor为0表示迭代完成
		if cursor == 0 {
			break
		}
	}

	return nil
}

func DelKeysWithSuffix(suffix string) error {
	// 使用Scan增量迭代，不会阻塞Redis，适合生产环境
	var cursor uint64
	for {
		keys, nextCursor, err := redisClient.Scan(ctx, cursor, "*"+suffix, 100).Result()
		if err != nil {
			return err
		}

		// 删除找到的键
		if len(keys) > 0 {
			if err := redisClient.Del(ctx, keys...).Err(); err != nil {
				return err
			}
			log.Println("成功删除相关后缀key", keys)
		}

		// 更新cursor继续迭代
		cursor = nextCursor
		// cursor为0表示迭代完成
		if cursor == 0 {
			break
		}
	}

	return nil
}

func DeleteAllRedisKeys() error {
	var cursor uint64 = 0
	for {
		keys, nextCursor, err := redisClient.Scan(ctx, cursor, "*", 0).Result()
		if err != nil {
			return err
		}
		cursor = nextCursor

		if len(keys) > 0 {
			_, err := redisClient.Del(ctx, keys...).Result()
			if err != nil {
				return err
			}
		}

		if cursor == 0 {
			break
		}
	}
	return nil
}