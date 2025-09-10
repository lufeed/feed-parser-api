package models

import "github.com/google/uuid"

type Source struct {
	ID          uuid.UUID `json:"id" `
	Name        string    `json:"name" `
	Description string    `json:"description" `
	FeedURL     string    `json:"feed_url"`
	HomeURL     string    `json:"home_url"`
	ImageURL    string    `json:"image_url"`
	IconURL     string    `json:"icon_url"`
	HTML        *string   `json:"html,omitempty"`
	UserID      string    `json:"user_id"`
	RequestID   string    `json:"request_id"`
}
