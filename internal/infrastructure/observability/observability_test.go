package observability

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEvaluateHealth_UPyDOWNPorDependencia(t *testing.T) {
	ok := func(context.Context) error { return nil }
	failing := func(context.Context) error { return errors.New("refused") }

	up := EvaluateHealth(context.Background(), "perfiles-service", map[string]Check{"database": ok, "messageBroker": ok})
	if up.Status != "UP" || up.Checks["database"] != "UP" || up.Checks["messageBroker"] != "UP" {
		t.Fatalf("report = %+v", up)
	}

	down := EvaluateHealth(context.Background(), "perfiles-service", map[string]Check{"database": failing, "messageBroker": ok})
	if down.Status != "DOWN" || down.Checks["database"] != "DOWN" || down.Checks["messageBroker"] != "UP" {
		t.Fatalf("report = %+v", down)
	}
}

func TestHealthHandler_Responde503SiAlgoEstaDown(t *testing.T) {
	handler := HealthHandler("perfiles-service", map[string]Check{
		"database": func(ctx context.Context) error {
			<-ctx.Done() // no responde: el timeout del check lo marca DOWN
			return ctx.Err()
		},
	})
	rec := httptest.NewRecorder()
	start := time.Now()
	handler(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	if time.Since(start) > 3*time.Second {
		t.Fatalf("el check no respetó su tiempo límite")
	}
	var report HealthReport
	if err := json.Unmarshal(rec.Body.Bytes(), &report); err != nil || report.Checks["database"] != "DOWN" {
		t.Fatalf("report = %+v, err = %v", report, err)
	}
}

func TestMetrics_PorPatronDeRutaSinIds(t *testing.T) {
	metrics := NewMetrics("perfiles-service")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /perfiles/{empleadoId}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("empleadoId") == "E404" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = io.WriteString(w, "{}")
	})
	mux.Handle("GET /metrics", metrics.Handler())
	server := metrics.Middleware(mux)

	for _, path := range []string{"/perfiles/E001", "/perfiles/E002", "/perfiles/E404", "/no-existe"} {
		server.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, path, nil))
	}
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rec.Body.String()

	for _, want := range []string{
		`http_requests_total{method="GET",route="/perfiles/{empleadoId}",service="perfiles-service",status="200"} 2`,
		`http_requests_total{method="GET",route="/perfiles/{empleadoId}",service="perfiles-service",status="404"} 1`,
		`route="unmatched"`,
		`http_request_duration_seconds_count{method="GET",route="/perfiles/{empleadoId}",service="perfiles-service",status="200"} 2`,
		`process_resident_memory_bytes`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("falta %q en /metrics", want)
		}
	}
	if strings.Contains(body, "E001") {
		t.Errorf("/metrics no debe contener ids concretos")
	}
}

func TestRouteTemplate(t *testing.T) {
	cases := map[string]string{
		"GET /perfiles/{empleadoId}": "/perfiles/{empleadoId}",
		"/perfiles":                  "/perfiles",
		"":                           UnmatchedRoute,
	}
	for pattern, want := range cases {
		if got := RouteTemplate(pattern); got != want {
			t.Errorf("RouteTemplate(%q) = %q, want %q", pattern, got, want)
		}
	}
}

func TestSetupTracing_SinEndpointNoExporta(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_ZIPKIN_ENDPOINT", "")
	t.Setenv("OTEL_SERVICE_NAME", "")
	shutdown, err := SetupTracing(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if ServiceName() != DefaultServiceName {
		t.Errorf("ServiceName() = %q", ServiceName())
	}
}

func TestSetupTracing_ConEndpointConfiguraElExportador(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_ZIPKIN_ENDPOINT", "http://127.0.0.1:1/api/v2/spans")
	t.Setenv("OTEL_SERVICE_NAME", "perfiles-service")
	shutdown, err := SetupTracing(context.Background())
	if err != nil {
		t.Fatalf("SetupTracing con endpoint: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = shutdown(ctx)
}
