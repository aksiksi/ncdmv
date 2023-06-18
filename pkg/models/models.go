// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.18.0

package models

import (
	"database/sql"
	"time"
)

type Appointment struct {
	ID              int64     `json:"id"`
	Location        string    `json:"location"`
	Time            time.Time `json:"time"`
	CreateTimestamp time.Time `json:"create_timestamp"`
}

type Notification struct {
	ID              int64          `json:"id"`
	AppointmentID   int64          `json:"appointment_id"`
	DiscordWebhook  sql.NullString `json:"discord_webhook"`
	CreateTimestamp time.Time      `json:"create_timestamp"`
}
