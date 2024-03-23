package models

import "time"

type CityScreening struct {
	ScreeningType string    `json:"screening_type" db:"screening_type"`
	TicketPrice   string    `json:"ticket_price" db:"ticket_price"`
	StartTime     time.Time `json:"start_time" db:"start_time"`
	ScreeningID   int64     `json:"id" db:"id"`
	HallID        int32     `json:"hall_id" db:"hall_id"`
	CinemaID      int32     `json:"cinema_id" db:"cinema_id"`
}
