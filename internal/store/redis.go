package store

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const rateLimitLua = `
local blocked = redis.call("EXISTS", KEYS[2])
if blocked == 1 then
	return 0
end

local current = redis.call("INCR", KEYS[1])

if current == 1 then
	redis.call("EXPIRE", KEYS[1], ARGV[1])
end

if current > tonumber(ARGV[2]) then
	redis.call("SET", KEYS[2], "1", "EX", ARGV[3])
	return 0
end

return 1
`

type RedisStore struct {
	client *redis.Client
	script *redis.Script
}

func NewRedisStore(addr, password string, db int) *RedisStore {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &RedisStore{
		client: client,
		script: redis.NewScript(rateLimitLua),
	}
}

func (s *RedisStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

func (s *RedisStore) Close() error {
	return s.client.Close()
}

func (s *RedisStore) Allow(
	ctx context.Context,
	key string,
	limit int,
	window time.Duration,
	blockDuration time.Duration,
) (bool, error) {
	countKey := "rate:" + key
	blockKey := "block:" + key

	result, err := s.script.Run(
		ctx,
		s.client,
		[]string{countKey, blockKey},
		int(window.Seconds()),
		limit,
		int(blockDuration.Seconds()),
	).Int()

	if err != nil {
		return false, err
	}

	return result == 1, nil
}
