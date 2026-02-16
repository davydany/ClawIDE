package model

import "time"

type Scratchpad struct {
	Content   string    `json:"content"`
	UpdatedAt time.Time `json:"updated_at"`
}
