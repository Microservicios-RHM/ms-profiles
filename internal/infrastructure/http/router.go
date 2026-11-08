package httpapi

import (
	"context"
	_ "embed"
	"log/slog"
	"net/http"
	"time"

	"github.com/Microservicios-RHM/ms-profiles/internal/application"
)

//go:embed openapi.json
var openAPIDocument []byte

//go:embed docs.html
var swaggerUIPage []byte

func NewRouter(
	getProfile *application.GetProfile,
	updateProfile *application.UpdateProfile,
	listProfiles *application.ListProfiles,
	healthCheck func(context.Context) map[string]string,
) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler(healthCheck))
	mux.HandleFunc("GET /metrics", metricsHandler)
	// Montadas bajo /perfiles (no en la raíz) para poder vivir detrás del API Gateway sin
	// reescritura de rutas: el Gateway ya proxea /perfiles/* preservando la ruta. El ServeMux de
	// Go 1.22+ prioriza el patrón literal más específico sobre {empleadoId}, así que no colisiona
	// con GET /perfiles/{empleadoId} sin importar el orden de registro.
	mux.HandleFunc("GET /perfiles/openapi.json", openAPIHandler)
	mux.HandleFunc("GET /perfiles/docs", swaggerUIHandler)
	mux.HandleFunc("GET /perfiles/docs/", swaggerUIHandler)

	mux.HandleFunc("GET /perfiles", listProfilesHandler(listProfiles))
	mux.HandleFunc("GET /perfiles/{empleadoId}", getProfileHandler(getProfile))
	mux.HandleFunc("PUT /perfiles/{empleadoId}", updateProfileHandler(updateProfile))

	return mux
}

func healthHandler(check func(context.Context) map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		checks := check(request.Context())
		status := "UP"
		for _, value := range checks {
			if value != "UP" {
				status = "DOWN"
				break
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if status == "DOWN" {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		_, _ = w.Write([]byte(`{"status":"` + status + `","service":"perfiles-service","checks":{"database":"` + checks["database"] + `","broker":"` + checks["broker"] + `"}}`))
	}
}

func metricsHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	_, _ = w.Write([]byte("# HELP rhm_profiles_service_info Static service identity metric\n# TYPE rhm_profiles_service_info gauge\nrhm_profiles_service_info{service=\"perfiles-service\"} 1\n"))
}

func openAPIHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(openAPIDocument)
}

func swaggerUIHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(swaggerUIPage)
}

// RequestLogger excluye /health desde el inicio (a diferencia de ms-notifications, donde tuvo que
// agregarse después porque el healthcheck de Docker cada pocos segundos ahogaba los logs reales).
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}
			start := time.Now()
			next.ServeHTTP(w, r)
			logger.Info(
				"request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"durationMs", time.Since(start).Milliseconds(),
			)
		})
	}
}
