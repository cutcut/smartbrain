package storage

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis"
	"time"
)

type Config struct {
	Addr     string
	Password string
	DB       int
}

type RedisStorage struct {
	config    Config
	client    *redis.Client
	resultTtl time.Duration
}

func (s *RedisStorage) Shutdown() error {
	return s.client.Close()
}

func InitStorage(config Config, resultTtl time.Duration) *RedisStorage {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	})

	if _, err := client.Ping().Result(); err != nil {
		panic(err)
	}

	return &RedisStorage{
		config:    config,
		client:    client,
		resultTtl: resultTtl,
	}
}

func getValuesKey(id string) string {
	return fmt.Sprintf("values_%s", id)
}

func getResultKey(id string) string {
	return fmt.Sprintf("result_%s", id)
}

func (s *RedisStorage) AddValue(id string, value Value) error {
	jsonValue, err := json.Marshal(value)

	if err != nil {
		return fmt.Errorf("[%s] Cannot marshal Value %s", id, err)
	}

	if status := s.client.LPush(getValuesKey(id), jsonValue); status.Err() != nil {
		return fmt.Errorf("[%s] Cannot push Value %s", id, status.Err())
	}

	return nil
}

func (s *RedisStorage) GetValues(id string) ([]Value, error) {
	values := s.client.LRange(getValuesKey(id), 0, -1)

	if values.Err() != nil {
		return []Value{}, fmt.Errorf("[%s] Cannot find Value %s", id, values.Err())
	}

	result := make([]Value, 0, len(values.Val()))

	for _, jsonValue := range values.Val() {
		value := Value{}
		err := json.Unmarshal([]byte(jsonValue), &value)

		if err != nil {
			return nil, fmt.Errorf("[%s] Cannot unmarshal Value %s %s", id, jsonValue, err)
		}

		result = append(result, value)
	}

	return result, nil
}

func (s *RedisStorage) DeleteValues(id string) error {
	status := s.client.Del(getValuesKey(id))

	if status.Err() != nil {
		return fmt.Errorf("[%s] Cannot delete Values %s", id, status.Err())
	}

	return nil
}

func (s *RedisStorage) AddResult(result Result) error {
	jsonResult, err := json.Marshal(result)

	if err != nil {
		return fmt.Errorf("[%s] Cannot marshal Result %v %s", result.Id, result.Values, err)
	}

	status := s.client.Set(getResultKey(result.Id), jsonResult, s.resultTtl)

	if status.Err() != nil {
		return fmt.Errorf("[%s] Cannot set Result %s", result.Id, status.Err())
	}

	return nil
}

func (s *RedisStorage) GetResult(id string) (Result, error) {
	jsonResult := s.client.Get(getResultKey(id))
	result := Result{}

	if jsonResult.Err() != nil {
		return result, fmt.Errorf("[%s] Cannot get Result %s", id, jsonResult.Err())
	}

	err := json.Unmarshal([]byte(jsonResult.Val()), &result)

	if err != nil {
		return result, fmt.Errorf("[%s] Cannot unmarshal Result %s %s", id, jsonResult.Val(), err)
	}

	return result, nil
}
