package observability

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// UnmatchedRoute etiqueta las peticiones que no coinciden con ninguna ruta registrada.
const UnmatchedRoute = "unmatched"

// Metrics expone métricas en formato Prometheus (client_golang) con los mismos nombres y
// etiquetas que los demás servicios del ecosistema, para que el dashboard use una sola consulta:
//   - http_requests_total{method, route, status}
//   - http_request_duration_seconds{method, route, status}  (_bucket, _sum, _count)
//   - process_cpu_seconds_total, process_resident_memory_bytes, go_*  (collectors estándar)
//
// `route` es el patrón del ServeMux (/perfiles/{empleadoId}), nunca la URL con el id real, para
// no crear una serie por empleado (alta cardinalidad).
type Metrics struct {
	registry *prometheus.Registry
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

func NewMetrics(service string) *Metrics {
	registry := prometheus.NewRegistry()
	registry.MustRegister(
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewGoCollector(),
	)
	constLabels := prometheus.Labels{"service": service}
	labels := []string{"method", "route", "status"}
	m := &Metrics{
		registry: registry,
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name:        "http_requests_total",
			Help:        "Total de peticiones HTTP recibidas, por método, ruta y código de respuesta",
			ConstLabels: constLabels,
		}, labels),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:        "http_request_duration_seconds",
			Help:        "Tiempo de respuesta de las peticiones HTTP en segundos",
			ConstLabels: constLabels,
			Buckets:     []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10},
		}, labels),
	}
	registry.MustRegister(m.requests, m.duration)
	return m
}

// Handler sirve GET /metrics.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{Registry: m.registry})
}

// Middleware mide cada petición. Debe envolver al ServeMux: el patrón de la ruta (r.Pattern) solo
// existe después de que el mux eligió el handler.
func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		labels := []string{r.Method, RouteTemplate(r.Pattern), strconv.Itoa(recorder.status)}
		m.requests.WithLabelValues(labels...).Inc()
		m.duration.WithLabelValues(labels...).Observe(time.Since(start).Seconds())
	})
}

// RouteTemplate quita el método del patrón del ServeMux ("GET /perfiles/{empleadoId}").
func RouteTemplate(pattern string) string {
	if pattern == "" {
		return UnmatchedRoute
	}
	if _, path, found := strings.Cut(pattern, " "); found {
		return path
	}
	return pattern
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(status int) {
	s.status = status
	s.ResponseWriter.WriteHeader(status)
}
