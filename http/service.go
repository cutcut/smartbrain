package http

import (
	"encoding/json"
	"fmt"
	"github.com/cutcut/smartbrain/tracker"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
	"net"
	"strings"
	"time"
)

type Service struct {
	trackerService *tracker.Service
	logger         *logrus.Entry
	port           uint
	host           string
}

func InitHttp(service *tracker.Service, logger *logrus.Entry, host string, port uint) Service {
	return Service{
		trackerService: service,
		logger:         logger,
		port:           port,
		host:           host,
	}
}

func (s Service) Start() {
	handler := func(ctx *fasthttp.RequestCtx) {
		s.logger.Infof("Http request: %s %s %s", ctx.Path(), ctx.PostBody(), ctx.QueryArgs())
		switch string(ctx.Path()) {
		case "/get-result":
			id := string(ctx.QueryArgs().Peek("id"))
			if id == "" {
				ctx.Error("Bad request:\nNot specified id", fasthttp.StatusBadRequest)
				return
			}

			result, err := s.trackerService.GetResult(id)

			if err != nil {
				ctx.Error("Not found", fasthttp.StatusNotFound)
				return
			}
			res, err := json.Marshal(result)

			if err != nil {
				s.logger.Errorf("Cannot marshal: %s", err)
			}

			ctx.Success("application/json", res)
		case "/new-tracker":
			period, durationErr := time.ParseDuration(string(ctx.PostArgs().Peek("period")))
			frequency, frequencyErr := time.ParseDuration(string(ctx.PostArgs().Peek("frequency")))

			var messages []string
			if frequencyErr != nil {
				messages = append(messages, "Not valid frequency")
			} else if frequency == 0 {
				messages = append(messages, "Frequency cannot be empty or zero")
			} else if time.Second > frequency || frequency > time.Hour {
				messages = append(messages, "Frequency cannot be less then 1 second & more then 1 hour")
			}
			if durationErr != nil {
				messages = append(messages, "Not valid period")
			} else if period == 0 {
				messages = append(messages, "Period cannot be empty or zero")
			} else if 2*time.Second > period || period > time.Hour {
				messages = append(messages, "Period cannot be less then 2 second & more then 1 hour")
			}

			if len(messages) == 0 && frequency > period {
				messages = append(messages, "Period cannot be less than frequency")
			}

			if len(messages) > 0 {
				ctx.Error("Bad request:\n"+strings.Join(messages, "\n"), fasthttp.StatusBadRequest)
				return
			}

			result, err := s.trackerService.NewTracker(period, frequency)

			if err != nil {
				ctx.Error("Error:\n"+err.Error(), fasthttp.StatusInternalServerError)
				return
			}

			ctx.Success("application/json", []byte("id:"+result))
		default:
			ctx.Error("Not found", fasthttp.StatusNotFound)
		}
	}

	server := &fasthttp.Server{Handler: handler}
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.host, s.port))

	if err != nil {
		panic(err)
	}

	go func() {
		if err := server.Serve(listener); err != nil {
			panic(err)
		}
	}()
}
