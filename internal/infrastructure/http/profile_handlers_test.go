package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Microservicios-RHM/ms-profiles/internal/application"
	"github.com/Microservicios-RHM/ms-profiles/internal/domain"
)

type fakeRepository struct {
	profiles map[string]domain.Profile
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{profiles: make(map[string]domain.Profile)}
}

func (r *fakeRepository) CreateIfNew(context.Context, string, domain.Profile) (bool, error) {
	return true, nil
}

func (r *fakeRepository) SyncIfNew(context.Context, string, string, string, string) (bool, bool, error) {
	return true, true, nil
}

func (r *fakeRepository) ArchiveIfNew(context.Context, string, string) (bool, bool, error) {
	return true, true, nil
}

func (r *fakeRepository) FindByEmployeeID(_ context.Context, empleadoID string) (domain.Profile, error) {
	if profile, ok := r.profiles[empleadoID]; ok {
		return profile, nil
	}
	return domain.Profile{}, domain.ErrProfileNotFound
}

func (r *fakeRepository) UpdateContactInfo(_ context.Context, empleadoID, telefono, direccion, ciudad, biografia string) (domain.Profile, error) {
	profile, ok := r.profiles[empleadoID]
	if !ok {
		return domain.Profile{}, domain.ErrProfileNotFound
	}
	profile.Telefono, profile.Direccion, profile.Ciudad, profile.Biografia = telefono, direccion, ciudad, biografia
	r.profiles[empleadoID] = profile
	return profile, nil
}

func (r *fakeRepository) ListAll(context.Context) ([]domain.Profile, error) {
	profiles := make([]domain.Profile, 0, len(r.profiles))
	for _, p := range r.profiles {
		profiles = append(profiles, p)
	}
	return profiles, nil
}

func buildTestRouter(repo *fakeRepository) http.Handler {
	return NewRouter(
		application.NewGetProfile(repo),
		application.NewUpdateProfile(repo),
		application.NewListProfiles(repo),
	)
}

func TestGetProfileHandler_Encontrado(t *testing.T) {
	repo := newFakeRepository()
	repo.profiles["E001"] = domain.Profile{
		ID: "p1", EmpleadoID: "E001", Nombre: "Juan", Email: "juan.perez@empresa.com",
		FechaCreacion: time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC),
	}
	router := buildTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/perfiles/E001", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("respuesta no es JSON válido: %v", err)
	}
	data := body["data"].(map[string]any)
	if data["empleadoId"] != "E001" {
		t.Errorf("empleadoId = %v, want E001", data["empleadoId"])
	}
	if _, hasSnakeCase := data["empleado_id"]; hasSnakeCase {
		t.Error("la respuesta no debe exponer snake_case")
	}
}

func TestGetProfileHandler_NoEncontrado(t *testing.T) {
	router := buildTestRouter(newFakeRepository())

	req := httptest.NewRequest(http.MethodGet, "/perfiles/E999", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	errInfo := body["error"].(map[string]any)
	if errInfo["code"] != "PROFILE_NOT_FOUND" {
		t.Errorf("error.code = %v, want PROFILE_NOT_FOUND", errInfo["code"])
	}
}

func TestUpdateProfileHandler_ActualizaCamposProvistos(t *testing.T) {
	repo := newFakeRepository()
	repo.profiles["E001"] = domain.Profile{EmpleadoID: "E001", Ciudad: "Bogotá"}
	router := buildTestRouter(repo)

	req := httptest.NewRequest(http.MethodPut, "/perfiles/E001", strings.NewReader(`{"ciudad":"Armenia"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if repo.profiles["E001"].Ciudad != "Armenia" {
		t.Errorf("ciudad no se actualizó: %+v", repo.profiles["E001"])
	}
}

func TestUpdateProfileHandler_NoEncontrado(t *testing.T) {
	router := buildTestRouter(newFakeRepository())

	req := httptest.NewRequest(http.MethodPut, "/perfiles/E999", strings.NewReader(`{"ciudad":"Armenia"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestUpdateProfileHandler_RechazaJSONMalformado(t *testing.T) {
	router := buildTestRouter(newFakeRepository())

	req := httptest.NewRequest(http.MethodPut, "/perfiles/E001", strings.NewReader(`{`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestUpdateProfileHandler_RechazaCamposNoSoportados(t *testing.T) {
	router := buildTestRouter(newFakeRepository())

	req := httptest.NewRequest(http.MethodPut, "/perfiles/E001", strings.NewReader(`{"nombre":"Otro"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (nombre no es editable por este endpoint)", rec.Code)
	}
}

func TestUpdateProfileHandler_RechazaTelefonoDemasiadoLargo(t *testing.T) {
	repo := newFakeRepository()
	repo.profiles["E001"] = domain.Profile{EmpleadoID: "E001"}
	router := buildTestRouter(repo)

	longPhone := strings.Repeat("1", 31)
	req := httptest.NewRequest(http.MethodPut, "/perfiles/E001", strings.NewReader(`{"telefono":"`+longPhone+`"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestListProfilesHandler_ListaVaciaCuandoNoHayPerfiles(t *testing.T) {
	router := buildTestRouter(newFakeRepository())

	req := httptest.NewRequest(http.MethodGet, "/perfiles", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	data := body["data"].([]any)
	if len(data) != 0 {
		t.Errorf("data = %v, want []", data)
	}
}

func TestHealthHandler(t *testing.T) {
	router := buildTestRouter(newFakeRepository())

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestOpenAPIHandler_SirveElDocumentoEmbebido(t *testing.T) {
	router := buildTestRouter(newFakeRepository())

	req := httptest.NewRequest(http.MethodGet, "/perfiles/openapi.json", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var doc map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("openapi.json no es JSON válido: %v", err)
	}
	if doc["openapi"] != "3.1.0" {
		t.Errorf("openapi = %v, want 3.1.0", doc["openapi"])
	}
}
