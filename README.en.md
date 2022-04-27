# Infrastructure

    docker-compose up

# Build

    go build -o ./bin/smartbrainsvc ./main.go 

# Run

    ./bin/smartbrainsvc
    export USE_HTTP=true && ./bin/smartbrainsvc # запуск с http сервером

# Http
    
    curl --location --request POST 'localhost:8080/new-tracker' \
    --header 'Content-Type: application/x-www-form-urlencoded' \
    --data-urlencode 'period=15s' \
    --data-urlencode 'frequency=1s'

    curl --location --request GET 'localhost:8080/get-result?id=240d13a7-943a-40e6-af57-4ef11738a957'

# Description

I understand the business need as tracking the bitcoin rate right now,
therefore, in the absence of handlers, the lifetime of the monitoring task is not extended in any way.

Asked at 15:00:00 to track 10 minutes with a frequency of 1 minute. This means the monitoring lifetime is up to 15:10:00.
If everything was downtime from 15:05:01, then 6 values that were received while the service was running would be available.

Restrictions:
- Frequency from 1 second to an hour
- Period from 2 seconds to an hour

The problem that if the service does not read from the queue, messages accumulate there, did not solve separately.
Сan be solved:
- Write messages to the queue with ttl (x-message-ttl for example an hour, taking into account the limitation on the period)
- Accumulate messages in the queue and as soon as the handler appears, consume them with some cheap processing (like now)

I made the assumption that monitoring results might not be stored permanently, so I chose Redis.

I chose RabbitMQ as the most suitable for distributed work.

The service has a limit on the number of simultaneous monitoring processes, hardcoded in code 2 for experiments.

### What I didn’t do, due to the fact that the time for a full-fledged resistant prod) solution takes more than a couple of days

I did not add logic with configs, I hardcoded everything in main. But the http server start flag (USE_HTTP),
so that you can run multiple instances of the service without changing the code.

Didn't use any frameworks for API, validated parameters with if-s. Surely there are better solutions.

bitcoin service:
For network calls to third-party services, I would generally add some reasonable timeouts for this external service.
Depending on the business logic, I would add some kind of fallback
requesting data from another provider of this data / previous bitcoin rate value from the cache

If the service can downtime for a long time, I would wrap it in a circuit breaker,
so that in case of problems, do not wait all the time allotted for a request for each request.

If the external service takes a long time to respond, then side effects are possible due to the fact that
that I have a minimum service polling frequency of 1 second, and I go for the value synchronously in the worker thread.
It would be possible to walk asynchronously, but then you need to write more logic for synchronizing the completion of monitoring and getting the result.

Since all workers essentially do the same thing and also send request the network for the same data,
in the general case, I would cache here locally (directly in the memory of the service, in order to get it very cheaply). But this is money.
I couldn't google how often the exchange rate changes
and whether there is at least some reasonable interval when it can be considered that the data has not had time to change.
Under highload, workers can simultaneously request the current bitcoin rate and
if this is acceptable for business, I would add a local cache in the service for at least 10ms, it is cheaper than going around the network.

Did not add metrics and monitoring for the service

Didn't write tests
