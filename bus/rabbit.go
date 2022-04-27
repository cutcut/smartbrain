package bus

import (
	"encoding/json"
	"fmt"
	"github.com/streadway/amqp"
	"time"
)

type Message struct {
	amqp.Delivery
}

func (d Message) Ack() {
	_ = d.Delivery.Ack(false)
}

func (d Message) Reject() {
	_ = d.Delivery.Reject(true)
}

type Config struct {
	Queue            string
	Url              string
	ReconnectTimeout time.Duration
}

type RabbitBus struct {
	config     Config
	connection *amqp.Connection
	channel    *amqp.Channel
	isStop     bool
}

func InitBus(config Config) *RabbitBus {
	rabbitBus := &RabbitBus{
		config: config,
	}

	if err := rabbitBus.connect(); err != nil {
		panic(err)
	}

	_, err := rabbitBus.channel.QueueDeclare(
		config.Queue,
		true,
		false,
		false,
		false,
		nil, // amqp.Table{"x-message-ttl": int32(time.Hour.Milliseconds())},
		// удалять то что заведомо уже не может быть обработано
	)

	if err != nil {
		panic(err)
	}

	return rabbitBus
}

func (s *RabbitBus) connect() error {
	var err error

	connection, err := amqp.Dial(s.config.Url)

	if err != nil {
		return err
	}

	channel, err := connection.Channel()

	if err != nil {
		return err
	}

	s.connection = connection
	s.channel = channel
	return nil
}

func (s *RabbitBus) Shutdown() error {
	s.isStop = true
	if err := s.channel.Close(); err != nil {
		return fmt.Errorf("cannot close channel: %s", err)
	}

	if err := s.connection.Close(); err != nil {
		return fmt.Errorf("cannot close connection: %s", err)
	}

	return nil
}

func (s *RabbitBus) Publish(tracker Tracker) error {
	value, err := json.Marshal(tracker)

	if err != nil {
		return fmt.Errorf("cannot marshal %s %s", tracker, err)
	}

	message := amqp.Publishing{
		ContentType: "text/plain",
		Body:        value,
	}

	if s.connection.IsClosed() {
		if err := s.connect(); err != nil {
			return err
		}
	}

	if err := s.channel.Publish("", s.config.Queue, false, false, message); err != nil {
		return fmt.Errorf("cannot publish %s %s", value, err)
	}

	return nil
}

func (s *RabbitBus) Consume() (chan AckTracker, chan error, error) {
	trackersCh := make(chan AckTracker)
	errorsCh := make(chan error)
	queueCh, err := s.getQueueChan()

	if err != nil {
		return nil, nil, fmt.Errorf("cannot get queue chan %s", err)
	}

	go func() {
		for {
			for msg := range queueCh {
				tracker := &Tracker{}

				if err := json.Unmarshal(msg.Body, tracker); err != nil {
					errorsCh <- fmt.Errorf("cannot unmarshal %s %s", string(msg.Body), err)
					continue
				}

				trackersCh <- AckTracker{*tracker, Message{msg}}
			}

			if s.isStop {
				break
			} else {
				errorsCh <- fmt.Errorf("connection closed unexpectedly")
				queueCh = s.reconnect(errorsCh)
			}
		}

		close(trackersCh)
		close(errorsCh)

	}()

	return trackersCh, errorsCh, nil
}

func (s *RabbitBus) getQueueChan() (<-chan amqp.Delivery, error) {
	return s.channel.Consume(
		s.config.Queue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
}

func (s *RabbitBus) reconnect(errorsCh chan error) <-chan amqp.Delivery {
	for {
		time.Sleep(s.config.ReconnectTimeout)

		if s.connection.IsClosed() {
			if err := s.connect(); err != nil {
				errorsCh <- fmt.Errorf("cannot connect %s", err)
				continue
			}
		}

		if queueChan, err := s.getQueueChan(); err != nil {
			errorsCh <- fmt.Errorf("cannot get queue chan %s", err)
		} else {
			errorsCh <- fmt.Errorf("reconnected success")
			return queueChan
		}
	}
}
