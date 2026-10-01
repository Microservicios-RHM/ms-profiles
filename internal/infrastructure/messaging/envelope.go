package messaging

// EventEnvelope es el espejo Go del envelope publicado por empleados-service/vacaciones-service.
// Ver docs/event-catalog.md en rhm-database-infrastructure.
type EventEnvelope struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Version    int            `json:"version"`
	OccurredAt string         `json:"occurredAt"`
	Producer   string         `json:"producer"`
	Data       map[string]any `json:"data"`
}

// StringField lee un campo string de Data de forma segura (nunca hace panic si falta o tiene otro tipo).
func (e EventEnvelope) StringField(key string) string {
	value, ok := e.Data[key].(string)
	if !ok {
		return ""
	}
	return value
}
