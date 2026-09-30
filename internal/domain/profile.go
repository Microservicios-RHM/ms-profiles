package domain

import "time"

// Profile es el perfil de un empleado. Archivado se agregó como campo propio (no está en el
// ejemplo JSON del enunciado) para que "archivar, no borrar" sea observable: sin él no habría
// forma de distinguir un perfil activo de uno archivado en la respuesta.
type Profile struct {
	ID            string
	EmpleadoID    string
	Nombre        string
	Email         string
	Telefono      string
	Direccion     string
	Ciudad        string
	Biografia     string
	Archivado     bool
	FechaCreacion time.Time
}
