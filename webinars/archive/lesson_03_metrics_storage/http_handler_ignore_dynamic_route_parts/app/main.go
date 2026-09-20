package main

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// httpRequestCounter Метрика для подсчета запросов по endpoints, сгруппированным по шаблонам
var httpRequestCounter = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests, grouped by route.",
	},
	[]string{"route", "method"},
)

// RoutePattern Структура для определения маппинга роута на статический путь
type RoutePattern struct {
	Regex      *regexp.Regexp
	StaticPath string // Статический путь для метрик (например, /user/:id -> /user/:id)
}

func createRoutePattern(pattern string) RoutePattern {
	// Заменяем параметры (например, :id, :name) на именованные группы захвата regex
	reString := regexp.QuoteMeta(pattern)
	param := regexp.MustCompile(`\:([a-zA-Z0-9_]+)`)
	reString = param.ReplaceAllString(reString, `(?P<$1>[a-zA-Z0-9_-]+)`)
	reString = "^" + reString + "$"

	log.Println(reString)

	re := regexp.MustCompile(reString)
	return RoutePattern{Regex: re, StaticPath: pattern} // статичный путь для метрики
}

var routePatterns = []RoutePattern{
	createRoutePattern("/user/:id"),
	createRoutePattern("/article/:article_id/comments"),
	createRoutePattern("/metrics"), // Не забыть самого себя добавить
	createRoutePattern("/"),
}

func matchRoute(path string) (RoutePattern, bool) {
	for _, pattern := range routePatterns {
		if pattern.Regex.MatchString(path) {
			return pattern, true
		}
	}
	return RoutePattern{}, false
}

// metricsMiddleware Middleware для обработки запросов и сбора метрик
func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		routePattern, found := matchRoute(r.URL.Path)
		route := r.URL.Path // По умолчанию используем оригинальный путь

		if found {
			route = routePattern.StaticPath // Используем статический путь из шаблона
			log.Printf("Matched route pattern: %s for path: %s\n", route, r.URL.Path)
		} else {
			log.Printf("No route pattern found for path: %s\n", r.URL.Path)
			route = "/not_found"
		}

		httpRequestCounter.With(prometheus.Labels{"route": route, "method": r.Method}).Inc()
		next.ServeHTTP(w, r)
	})
}

// Обработчик для /user/:id
func userHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/user/")
	id, err := strconv.Atoi(idStr)

	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, "User ID: %d\n", id)
}

// Обработчик для /article/:article_id/comments
func articleCommentsHandler(w http.ResponseWriter, r *http.Request) {
	articleID := strings.Split(strings.TrimPrefix(r.URL.Path, "/article/"), "/")[0] // Получаем только ID статьи
	fmt.Fprintf(w, "Article ID: %s - Showing comments\n", articleID)
}

// Обработчик для /
func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome!\n")
}

func main() {
	// Создаем обработчики
	http.HandleFunc("/user/", userHandler)
	http.HandleFunc("/article/", articleCommentsHandler)
	http.HandleFunc("/", rootHandler)

	// Применяем middleware для сбора метрик
	handler := metricsMiddleware(http.DefaultServeMux)

	http.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{},
	))

	fmt.Println("Server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
