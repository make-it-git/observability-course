package main

import (
	"errors"
	"log"
	"math/rand"
	"time"

	"github.com/sony/gobreaker"
)

var ErrExternalService = errors.New("Error calling external service")
var errorPropability = 0.5

func callExternalService() (string, error) {
	if rand.Float64() < errorPropability {
		return "", ErrExternalService
	}

	return "result", nil
}

func main() {
	var cb *gobreaker.CircuitBreaker
	settings := gobreaker.Settings{
		Name:        "external_service",
		MaxRequests: 1,           // Max number of requests when half-open
		Interval:    time.Second, // Time until switch to half-open
		Timeout:     time.Second, // Time to wait for a response before tripping
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			log.Printf("Failure ratio %v", failureRatio)
			return counts.Requests > 10 && failureRatio >= 0.3
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			log.Printf("Circuit Breaker '%s' changed from '%s' to '%s'\n", name, from, to)
		},
	}
	cb = gobreaker.NewCircuitBreaker(settings)

	for i := 0; i < 50; i++ {
		result, err := cb.Execute(func() (interface{}, error) {
			return callExternalService()
		})

		if err != nil {
			log.Printf("Request %d failed: %v\n", i, err)
		} else {
			log.Printf("Request %d successful: %s\n", i, result)
		}
		time.Sleep(time.Millisecond * 50)
	}

	log.Printf("*******")
	time.Sleep(time.Second * 2)

	for i := 0; i < 50; i++ {
		result, err := cb.Execute(func() (interface{}, error) {
			return callExternalService()
		})

		if err != nil {
			log.Printf("Request %d failed: %v\n", i, err)
		} else {
			log.Printf("Request %d successful: %s\n", i, result)
		}
		time.Sleep(time.Millisecond * 50)
	}
}
