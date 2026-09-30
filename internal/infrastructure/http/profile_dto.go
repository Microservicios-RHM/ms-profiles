package httpapi

import "github.com/Microservicios-RHM/ms-profiles/internal/domain"

// profileDTO expone el perfil en camelCase, igual que el resto del ecosistema (fechaEnvio,
// empleadoId en ms-notifications; aquí empleadoId, fechaCreacion).
type profileDTO struct {
	ID            string `json:"id"`
	EmpleadoID    string `json:"empleadoId"`
	Nombre        string `json:"nombre"`
	Email         string `json:"email"`
	Telefono      string `json:"telefono"`
	Direccion     string `json:"direccion"`
	Ciudad        string `json:"ciudad"`
	Biografia     string `json:"biografia"`
	Archivado     bool   `json:"archivado"`
	FechaCreacion string `json:"fechaCreacion"`
}

func toDTO(p domain.Profile) profileDTO {
	return profileDTO{
		ID:            p.ID,
		EmpleadoID:    p.EmpleadoID,
		Nombre:        p.Nombre,
		Email:         p.Email,
		Telefono:      p.Telefono,
		Direccion:     p.Direccion,
		Ciudad:        p.Ciudad,
		Biografia:     p.Biografia,
		Archivado:     p.Archivado,
		FechaCreacion: p.FechaCreacion.UTC().Format("2006-01-02T15:04:05.000Z"),
	}
}

func toDTOList(profiles []domain.Profile) []profileDTO {
	dtos := make([]profileDTO, 0, len(profiles))
	for _, p := range profiles {
		dtos = append(dtos, toDTO(p))
	}
	return dtos
}
