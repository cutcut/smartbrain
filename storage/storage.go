package storage

import "time"

type Result struct {
	Id     string  `json:"Id"`
	Values []Value `json:"Values"`
}

type Value struct {
	Time  time.Time `json:"Time"`
	Value float32   `json:"Value"`
}

type Storage interface {
	AddValue(id string, value Value) error
	GetValues(id string) ([]Value, error)
	DeleteValues(id string) error
	AddResult(result Result) error
	GetResult(id string) (Result, error)
}
