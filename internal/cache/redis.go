package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
	ctx    context.Context
}

type CacheConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

func NewRedisClient(config *CacheConfig) *RedisClient {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", config.Host, config.Port),
		Password: config.Password,
		DB:       config.DB,
	})

	return &RedisClient{
		client: rdb,
		ctx:    context.Background(),
	}
}

func (r *RedisClient) Ping() error {
	_, err := r.client.Ping(r.ctx).Result()
	return err
}

func (r *RedisClient) Set(key string, value interface{}, ttl time.Duration) error {
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return r.client.Set(r.ctx, key, jsonValue, ttl).Err()
}

func (r *RedisClient) Get(key string, dest interface{}) error {
	val, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(val), dest)
}

func (r *RedisClient) Delete(key string) error {
	return r.client.Del(r.ctx, key).Err()
}

func (r *RedisClient) DeletePattern(pattern string) error {
	keys, err := r.client.Keys(r.ctx, pattern).Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		return r.client.Del(r.ctx, keys...).Err()
	}

	return nil
}

func (r *RedisClient) Exists(key string) (bool, error) {
	result, err := r.client.Exists(r.ctx, key).Result()
	return result > 0, err
}

func (r *RedisClient) SetTTL(key string, ttl time.Duration) error {
	return r.client.Expire(r.ctx, key, ttl).Err()
}

func (r *RedisClient) GetTTL(key string) (time.Duration, error) {
	return r.client.TTL(r.ctx, key).Result()
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}

// Cache key generators
func TechnicianCacheKey(page, size int, search string) string {
	if search != "" {
		return fmt.Sprintf("technicians:search:%s:page:%d:size:%d", search, page, size)
	}
	return fmt.Sprintf("technicians:list:page:%d:size:%d", page, size)
}

func TechnicianDetailCacheKey(id string) string {
	return fmt.Sprintf("technician:detail:%s", id)
}

func TechniciansByCityCacheKey(city string) string {
	return fmt.Sprintf("technicians:city:%s", city)
}

func TechniciansByStateCacheKey(state string) string {
	return fmt.Sprintf("technicians:state:%s", state)
}

func DashboardCacheKey() string {
	return "dashboard:stats"
}

func ClientsCacheKey(page, size int) string {
	return fmt.Sprintf("clients:list:page:%d:size:%d", page, size)
}

// Cache TTL constants
const (
	TechnicianListTTL    = 5 * time.Minute   // Lista de técnicos
	TechnicianDetailTTL  = 10 * time.Minute  // Detalhes do técnico
	TechnicianSearchTTL  = 2 * time.Minute   // Busca de técnicos
	TechnicianFilterTTL  = 3 * time.Minute   // Filtros por cidade/estado
	DashboardTTL         = 1 * time.Minute   // Dashboard stats
	ClientsTTL           = 5 * time.Minute   // Lista de clientes
	DefaultTTL           = 10 * time.Minute  // TTL padrão
)