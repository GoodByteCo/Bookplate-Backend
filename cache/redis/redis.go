package redis

import (
	"os"
	"time"

	"github.com/go-redis/redis/v7"
)

var prefix string = "_PAGE_CACHE_"

type Storage struct {
	client *redis.Client
}

func (s Storage) Get(key string) []byte {
	val, _ := s.client.Get(prefix + key).Bytes()
	return val
}

func (s Storage) Set(key string, content []byte, duration time.Duration) {
	s.client.Set(prefix+key, content, duration)
}

func NewStorage() *Storage {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6377",
		Password: os.Getenv("REDISPWD"),
		DB:       0,
	})
	storage := Storage{
		client: client,
	}
	return &storage
}

//todo actully deal wth errors
