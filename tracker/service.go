package tracker

import (
	"context"
	"fmt"
	"github.com/cutcut/smartbrain/bitcoin"
	"github.com/cutcut/smartbrain/bus"
	"github.com/cutcut/smartbrain/storage"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"time"
)

type Service struct {
	logger     *logrus.Entry
	storage    storage.Storage
	bus        bus.Bus
	bitcoin    bitcoin.Service
	maxWorkers uint
}

func InitService(logger *logrus.Entry, storage storage.Storage, bus bus.Bus, maxWorkers uint) *Service {
	return &Service{
		logger:     logger,
		storage:    storage,
		bus:        bus,
		maxWorkers: maxWorkers,
	}
}

func (s *Service) Start(ctx context.Context) {
	workers := make(chan interface{}, s.maxWorkers)
	trackersCh, errorsCh, err := s.bus.Consume()

	if err != nil {
		panic(err)
	}

	go func() {
		for err := range errorsCh {
			s.logger.Errorf("Consume error: %s", err)
		}
	}()

	go func() {
		for tracker := range trackersCh {
			workers <- nil
			go func(ackTracker bus.AckTracker) {
				s.logger.Infof("[%s] CONSUME %v", ackTracker.Id, ackTracker.Tracker)

				if s.process(ctx, ackTracker.Tracker) {
					ackTracker.Ack()
					s.logger.Infof("[%s] ACK %v", ackTracker.Id, ackTracker.Tracker)
				} else {
					ackTracker.Reject()
					s.logger.Infof("[%s] REJECT %v", ackTracker.Id, ackTracker.Tracker)
				}

				<-workers
			}(tracker)
		}
	}()
}

func (s *Service) GetResult(id string) (storage.Result, error) {
	result, err := s.storage.GetResult(id)
	if err != nil {
		s.logger.Errorf("Cannot get Result: %s", err)
	}
	return result, err
}

func (s *Service) NewTracker(period time.Duration, frequency time.Duration) (string, error) {
	tracker := bus.Tracker{
		Id:        uuid.New().String(),
		Frequency: frequency,
		Expire:    time.Now().Add(period).Add(time.Second),
	}

	if err := s.bus.Publish(tracker); err != nil {
		s.logger.Errorf("Cannot publish Tracker: %s", err)
		return "", fmt.Errorf("[%s] Cannot publish %v %s", tracker.Id, tracker, err)
	}

	s.logger.Infof("[%s] PUBLISHED: %v", tracker.Id, tracker)

	return tracker.Id, nil
}

func (s *Service) process(ctx context.Context, tracker bus.Tracker) bool {
	if tracker.Expire.Before(time.Now()) {
		return s.logDelete(tracker)
	}

	s.logGenerate(tracker)

	for {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(tracker.Expire.Sub(time.Now())):
			return s.logDelete(tracker)
		case <-time.After(tracker.Frequency):
			s.logGenerate(tracker)
		}
	}
}

func (s *Service) logDelete(tracker bus.Tracker) bool {
	if err := s.delete(tracker); err != nil {
		s.logger.Errorf("%s", err)
		return false
	} else {
		return true
	}
}

func (s *Service) logGenerate(tracker bus.Tracker) {
	if err := s.generate(tracker); err != nil {
		s.logger.Errorf("%s", err)
	}
}

func (s *Service) delete(tracker bus.Tracker) error {
	values, err := s.storage.GetValues(tracker.Id)

	if err != nil {
		return fmt.Errorf("[%s] Cannot get Values %v %s", tracker.Id, tracker, err)
	}

	result := storage.Result{
		Id:     tracker.Id,
		Values: values,
	}

	if err := s.storage.AddResult(result); err != nil {
		return fmt.Errorf("[%s] Cannot add Result %v %s", result.Id, result, err)
	}

	s.logger.Infof("[%s] STORAGED %v", result.Id, result)

	if err := s.storage.DeleteValues(tracker.Id); err != nil {
		return fmt.Errorf("[%s] Cannot delete Values %v %s", result.Id, result, err)
	}

	s.logger.Infof("[%s] DELETED %v", tracker.Id, tracker)

	return nil
}

func (s *Service) generate(tracker bus.Tracker) error {
	response, err := s.bitcoin.GetAmount()

	if err != nil {
		return fmt.Errorf("[%s] Cannot get Response %v %s", tracker.Id, tracker, err)
	}

	value := storage.Value{
		Time:  time.Now(),
		Value: response.Amount,
	}

	if err = s.storage.AddValue(tracker.Id, value); err != nil {
		return fmt.Errorf("[%s] Cannot add Value %v %s", tracker.Id, tracker, err)
	}

	s.logger.Infof("[%s] GENERATED %v %v", tracker.Id, tracker, value)

	return nil
}
