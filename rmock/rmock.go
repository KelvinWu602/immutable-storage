// An Redis implementation of the ImmutableStorage interface.
package rmock

import (
	"context"
	"encoding/base64"
	"log"
	"time"

	"github.com/KelvinWu602/immutable-storage/blueprint"
	"github.com/redis/go-redis/v9"
)

// IPFS implements blueprint.ImmutableStorage.
type REDIS struct {
	redisClient *redis.Client
}

func New() *REDIS {
	var r REDIS
	r.redisClient = redis.NewClient(&redis.Options{
		Addr:     "13.214.28.186:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	return &r
}

func (r *REDIS) Store(key blueprint.Key, message []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	keyBase64Url := base64.URLEncoding.EncodeToString(key[:])
	msgBase64Std := base64.StdEncoding.EncodeToString(message)
	err := r.redisClient.Set(ctx, keyBase64Url, msgBase64Std, 0).Err()
	return err
}

func (r *REDIS) Read(key blueprint.Key) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	keyBase64Url := base64.URLEncoding.EncodeToString(key[:])
	val, err := r.redisClient.Get(ctx, keyBase64Url).Result()
	if err != nil {
		return nil, err
	}
	message, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		return nil, err
	}
	return message[48:], nil
}

func (r *REDIS) AvailableKeys() []blueprint.Key {
	keyBase64Url := []string{}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	iter := r.redisClient.Scan(ctx, 0, "*", 1000).Iterator()
	for iter.Next(ctx) {
		log.Println("iter.val", iter.Val())
		keyBase64Url = append(keyBase64Url, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return nil
	}

	keys := []blueprint.Key{}
	for _, keyb64 := range keyBase64Url {
		log.Println(keyb64)
		key, err := base64.URLEncoding.DecodeString(keyb64)
		if err != nil {
			continue
		}
		keys = append(keys, blueprint.Key(key))
	}
	return keys
}

func (r *REDIS) IsDiscovered(key blueprint.Key) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	keyBase64Url := base64.URLEncoding.EncodeToString(key[:])
	val, err := r.redisClient.Get(ctx, keyBase64Url).Result()
	if err != nil {
		return false
	}
	if _, err = base64.StdEncoding.DecodeString(val); err != nil {
		return false
	}
	return true
}
