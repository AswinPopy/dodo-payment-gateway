package model

import "time"

type APIKey struct {
	ID         string     `json:"id"`
	BusinessID string     `json:"business_id"`
	KeyHash    string     `json:"-"`
	CreatedAt  time.Time  `json:"created_at"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
}
