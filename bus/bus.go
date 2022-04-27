package bus

import (
	"time"
)

type Acknowledger interface {
	Ack()
	Reject()
}

type Tracker struct {
	Id        string        `json:"Id"`
	Frequency time.Duration `json:"Frequency"`
	Expire    time.Time     `json:"Expire"`
}

type AckTracker struct {
	Tracker
	Acknowledger
}

type Bus interface {
	Publish(tracker Tracker) error
	Consume() (chan AckTracker, chan error, error)
}
