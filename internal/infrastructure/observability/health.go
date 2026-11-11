package observability

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Check verifica una dependencia; un error o false significa DOWN.
type Check func(ctx context.Context) error

// HealthReport es el cuerpo de GET /health (Reto 8):
//
//	200 {"status":"UP","service":"perfiles-service","checks":{"database":"UP","messageBroker":"UP"}}
//	503 {"status":"DOWN", ... }
type HealthReport struct {
	Status  string            `json:"status"`
	Service string            `json:"service"`
	Checks  map[string]string `json:"checks"`
}

const checkTimeout = 2 * time.Second

// EvaluateHealth ejecuta los checks en paralelo, cada uno con un tiempo límite corto: un health
// check lento degrada al propio servicio.
func EvaluateHealth(ctx context.Context, service string, checks map[string]Check) HealthReport {
	report := HealthReport{Status: "UP", Service: service, Checks: make(map[string]string, len(checks))}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for name, check := range checks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			checkCtx, cancel := context.WithTimeout(ctx, checkTimeout)
			defer cancel()
			status := "UP"
			if err := check(checkCtx); err != nil {
				status = "DOWN"
			}
			mu.Lock()
			report.Checks[name] = status
			if status == "DOWN" {
				report.Status = "DOWN"
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	return report
}

// HealthHandler sirve GET /health: 200 si todo está UP, 503 si alguna dependencia está DOWN.
func HealthHandler(service string, checks map[string]Check) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		report := EvaluateHealth(r.Context(), service, checks)
		w.Header().Set("Content-Type", "application/json")
		if report.Status == "UP" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		_ = json.NewEncoder(w).Encode(report)
	}
}
