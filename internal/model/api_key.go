package model

import "time"

type APIKey struct {
	ID         string     `json:"id"`
	BusinessID string     `json:"business_id"`
	Name       *string    `json:"name,omitempty"`
	KeyPrefix  string     `json:"key_prefix"`
	KeyHash    string     `json:"-"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
}
