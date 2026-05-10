package domain

import "time"

type Item struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Category  string    `json:"category"`
	Score     float64   `json:"score"`
	Tags      []string  `json:"tags"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
type AuditEvent struct {
	EventID   string    `json:"event_id"`
	ItemID    string    `json:"item_id"`
	Action    string    `json:"action"` // "created", "updated", "deleted"
	ActorID   string    `json:"actor_id"`
	Timestamp time.Time `json:"timestamp"`
	Payload   string    `json:"payload"` // JSON-serialized delta
}
