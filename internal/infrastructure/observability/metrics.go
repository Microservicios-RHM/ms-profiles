package observability

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const ServiceName = "perfiles-service"

var metrics = struct {
	sync.Mutex
	requests  map[string]uint64
	durations map[string]float64
	events    map[string]uint64
}{requests: map[string]uint64{}, durations: map[string]float64{}, events: map[string]uint64{}}

func ObserveHTTP(method string, status int, duration time.Duration) {
	metrics.Lock()
	defer metrics.Unlock()
	metrics.requests[fmt.Sprintf("%s|%d", method, status)]++
	metrics.durations[method] += duration.Seconds()
}

func ObserveEvent(eventType, outcome string) {
	metrics.Lock()
	defer metrics.Unlock()
	metrics.events[eventType+"|"+outcome]++
}

func RenderMetrics() string {
	metrics.Lock()
	defer metrics.Unlock()
	lines := []string{"# HELP rhm_service_info Static service identity metric", "# TYPE rhm_service_info gauge", `rhm_service_info{service="` + ServiceName + `"} 1`, "# HELP rhm_http_requests_total HTTP requests by method and status", "# TYPE rhm_http_requests_total counter"}
	keys := func(m map[string]uint64) []string {
		out := make([]string, 0, len(m))
		for k := range m {
			out = append(out, k)
		}
		sort.Strings(out)
		return out
	}
	for _, key := range keys(metrics.requests) {
		parts := strings.Split(key, "|")
		lines = append(lines, fmt.Sprintf(`rhm_http_requests_total{service="%s",method="%s",status="%s"} %d`, ServiceName, parts[0], parts[1], metrics.requests[key]))
	}
	lines = append(lines, "# HELP rhm_http_request_duration_seconds_sum Cumulative HTTP request duration", "# TYPE rhm_http_request_duration_seconds_sum counter")
	for method, value := range metrics.durations {
		lines = append(lines, fmt.Sprintf(`rhm_http_request_duration_seconds_sum{service="%s",method="%s"} %f`, ServiceName, method, value))
	}
	lines = append(lines, "# HELP rhm_events_total Events handled by outcome", "# TYPE rhm_events_total counter")
	for _, key := range keys(metrics.events) {
		parts := strings.Split(key, "|")
		lines = append(lines, fmt.Sprintf(`rhm_events_total{service="%s",event_type="%s",outcome="%s"} %d`, ServiceName, parts[0], parts[1], metrics.events[key]))
	}
	return strings.Join(lines, "\n") + "\n"
}
