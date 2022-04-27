package bitcoin

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

type Response struct {
	Amount float32 `json:"amount"`
}

type Service struct {
}

func (s Service) GetAmount() (*Response, error) {
	// TODO circuit breaker, fallback, timeout
	jsonResponse := genRsp()
	response := Response{}
	err := json.Unmarshal([]byte(jsonResponse), &response)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal result %s %s", jsonResponse, err)
	}
	return &response, nil
}

func genRsp() string {
	rand.Seed(time.Now().UnixNano())
	min := 39000
	max := 41000
	amount := rand.Intn(max-min) + min
	return fmt.Sprintf(`{ "amount": %d }`, amount)
}
