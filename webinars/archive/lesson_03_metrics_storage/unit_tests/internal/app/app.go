package app

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/mocks.go -package=mocks
type PrometheusClient interface {
	Inc(label1, label2 string)
}

type Service struct {
	client  PrometheusClient
	counter *prometheus.CounterVec
}

var counter *prometheus.CounterVec

func init() {
	counter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "my_counter",
			Help: "A simple counter metric.",
		},
		[]string{"label1", "label2"},
	)
	prometheus.MustRegister(counter)
}

func New(client PrometheusClient) *Service {

	return &Service{
		client:  client,
		counter: counter,
	}
}

func (m *Service) DoStuff(label1, label2 string) {
	fmt.Println("Important work")
	m.client.Inc(label1, label2)
}
