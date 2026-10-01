package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/Microservicios-RHM/ms-profiles/internal/application"
	"github.com/Microservicios-RHM/ms-profiles/internal/domain"
)

type updateProfileRequest struct {
	Telefono  *string `json:"telefono"`
	Direccion *string `json:"direccion"`
	Ciudad    *string `json:"ciudad"`
	Biografia *string `json:"biografia"`
}

func listProfilesHandler(uc *application.ListProfiles) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profiles, err := uc.Execute(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Error interno del servidor", errCodeInternal, r.URL.Path)
			return
		}
		writeSuccess(w, http.StatusOK, "Perfiles consultados correctamente", toDTOList(profiles))
	}
}

func getProfileHandler(uc *application.GetProfile) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		empleadoID := r.PathValue("empleadoId")

		profile, err := uc.Execute(r.Context(), empleadoID)
		if errors.Is(err, domain.ErrProfileNotFound) {
			writeError(
				w, http.StatusNotFound,
				fmt.Sprintf("El perfil del empleado %s no existe", empleadoID),
				errCodeProfileNotFound, r.URL.Path,
			)
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Error interno del servidor", errCodeInternal, r.URL.Path)
			return
		}
		writeSuccess(w, http.StatusOK, "Perfil consultado correctamente", toDTO(profile))
	}
}

func updateProfileHandler(uc *application.UpdateProfile) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		empleadoID := r.PathValue("empleadoId")

		var body updateProfileRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "El cuerpo no es JSON válido", errCodeInvalidJSON, r.URL.Path)
			return
		}

		if details := validateUpdateRequest(body); len(details) > 0 {
			writeError(w, http.StatusBadRequest, "Datos de entrada inválidos", errCodeValidation, r.URL.Path, details...)
			return
		}

		profile, err := uc.Execute(r.Context(), empleadoID, application.UpdateProfileInput{
			Telefono:  body.Telefono,
			Direccion: body.Direccion,
			Ciudad:    body.Ciudad,
			Biografia: body.Biografia,
		})
		if errors.Is(err, domain.ErrProfileNotFound) {
			writeError(
				w, http.StatusNotFound,
				fmt.Sprintf("El perfil del empleado %s no existe", empleadoID),
				errCodeProfileNotFound, r.URL.Path,
			)
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Error interno del servidor", errCodeInternal, r.URL.Path)
			return
		}
		writeSuccess(w, http.StatusOK, "Perfil actualizado correctamente", toDTO(profile))
	}
}

// validateUpdateRequest respeta los límites de columna de la migración (VARCHAR(30)/(255)/(100));
// biografia es TEXT sin límite, así que no se valida.
func validateUpdateRequest(body updateProfileRequest) []errorDetail {
	var details []errorDetail
	if body.Telefono != nil && len(*body.Telefono) > 30 {
		details = append(details, errorDetail{Field: "telefono", Message: "telefono no puede superar 30 caracteres"})
	}
	if body.Direccion != nil && len(*body.Direccion) > 255 {
		details = append(details, errorDetail{Field: "direccion", Message: "direccion no puede superar 255 caracteres"})
	}
	if body.Ciudad != nil && len(*body.Ciudad) > 100 {
		details = append(details, errorDetail{Field: "ciudad", Message: "ciudad no puede superar 100 caracteres"})
	}
	return details
}
