package model

import "time"

type Business struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	WebhookURL    *string   `json:"webhook_url,omitempty"`
	WebhookSecret string    `json:"webhook_secret,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}
