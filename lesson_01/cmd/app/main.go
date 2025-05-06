// Setup pyroscope
package main

import (
	"context"
	"fmt"
	metrics "github.com/hashicorp/go-metrics"
	"github.com/marpaia/graphite-golang"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/grafana/pyroscope-go"
	loki "github.com/magnetde/loki"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/sirupsen/logrus"
)

var (
	requestCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
	)
	duration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Histogram of response time for handler",
			Buckets: prometheus.DefBuckets,
		},
	)
	pushGatewayAddr = os.Getenv("PROM_PUSHGATEWAY_ADDR")
)

func init() {
	prometheus.MustRegister(requestCount)
	prometheus.MustRegister(duration)
}

func handler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestCount.Inc()
	fmt.Fprintf(w, "Hello, World!")
	n := rand.Intn(10)
	time.Sleep(time.Millisecond * time.Duration(n))
	duration.Observe(time.Since(start).Seconds())
	logrus.WithFields(logrus.Fields{
		"path":   "/",
		"method": "GET",
		"status": 200,
	}).Info("Request completed")
}

func pushMetrics() {
	pusher := push.New(pushGatewayAddr, "example_push").Collector(requestCount)
	err := pusher.Push()
	if err != nil {
		log.Println("Error pushing metrics to Pushgateway:", err)
	} else {
		log.Println("Push completed")
	}
}

func sendToGraphite() {
	graphiteHost := os.Getenv("GRAPHITE_HOST")
	graphitePort, _ := strconv.ParseInt(os.Getenv("GRAPHITE_PORT"), 10, 64)
	graphite, err := graphite.NewGraphite(graphiteHost, int(graphitePort))
	if err != nil {
		log.Println("Error connecting to Graphite:", err)
		return
	}
	defer graphite.Disconnect()
	err = graphite.SimpleSend("golang_observability.request_count", "1")
	if err != nil {
		log.Println("Error sending metric to Graphite:", err)
	}
	log.Println("graphite send completed")
}

func slowMethod() {
	defer metrics.MeasureSince([]string{"SlowMethod"}, time.Now())

	time.Sleep(time.Second * 3)

	//runtime.SetMutexProfileFraction(5)
	//time.Sleep(time.Millisecond)
	//i := 0
	//
	//m := sync.Mutex{}
	//wg := sync.WaitGroup{}
	//wg.Add(2)
	//go func() {
	//	defer wg.Done()
	//	for i < 10000 {
	//		m.Lock()
	//		i++
	//		m.Unlock()
	//	}
	//}()
	//go func() {
	//	defer wg.Done()
	//	for i < 10000 {
	//		m.Lock()
	//		i++
	//		m.Unlock()
	//	}
	//}()
	//wg.Wait()

	metrics.IncrCounter([]string{"hello"}, 42)
}

func configureStatsd() {
	graphiteHost := os.Getenv("STATSD_HOST")
	graphitePort, _ := strconv.ParseInt(os.Getenv("STATSD_PORT"), 10, 64)
	sink, err := metrics.NewStatsdSink(fmt.Sprintf("%s:%d", graphiteHost, graphitePort))
	if err != nil {
		log.Fatal(err)
	}
	metrics.NewGlobal(metrics.DefaultConfig("my-app"), sink)
}

func main() {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)

	hook := loki.NewHook(fmt.Sprintf("http://%s", os.Getenv("LOKI_HOST")), loki.WithName("my-awesome-app"), loki.WithLabel("env", "dev"))
	defer hook.Close()
	logger.AddHook(hook)
	logger.Info("HELLO")
	logger.Debug("HELLO DEBUG")

	configureStatsd()

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", handler)

	pyroscope.Start(pyroscope.Config{
		ApplicationName: "my.golang.app",
		ServerAddress:   os.Getenv("PYROSCOPE_ADDR"),
		Logger:          pyroscope.StandardLogger,
		Tags:            map[string]string{"hostname": os.Getenv("HOSTNAME")},
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,
			pyroscope.ProfileGoroutines,
			pyroscope.ProfileMutexCount,
			pyroscope.ProfileMutexDuration,
			pyroscope.ProfileBlockCount,
			pyroscope.ProfileBlockDuration,
		},
	})

	go func() {
		for {
			time.Sleep(time.Second)
			pushMetrics()
			sendToGraphite()
		}
	}()

	go func() {
		for {
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", "slowMethod"), func(c context.Context) {
				slowMethod()
			})
		}
	}()

	logger.Info("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Fatal("Error starting HTTP server: ", err)
	}
}
