package domain

import "errors"

// ErrProfileNotFound es el error de dominio para "el perfil de este empleado no existe todavía".
// La capa HTTP lo traduce a 404; los handlers de eventos lo tratan como una advertencia (no
// debería ocurrir en operación normal, ver notify_profile_synced.go / archive_profile.go).
var ErrProfileNotFound = errors.New("perfil no encontrado")
