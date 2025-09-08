package models

import (
	"github.com/google/uuid"
	"time"
)

type Feed struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	ImageURL    string    `json:"image_url"`
	HTML        *string   `json:"html,omitempty"`
	PublishedAt time.Time `json:"published_at"`
	FeedID      string    `json:"feed_id"`
	FeedName    string    `json:"feed_name"`
	UserID      string    `json:"user_id"`
}
