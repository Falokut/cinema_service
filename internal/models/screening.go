package models

import "time"

type Screening struct {
	ScreeningID   int64     `json:"id" db:"id"`
	ScreeningType string    `json:"screening_type" db:"screening_type"`
	TicketPrice   string    `json:"ticket_price" db:"ticket_price"`
	StartTime     time.Time `json:"start_time" db:"start_time"`
	HallID        int32     `json:"hall_id" db:"hall_id"`
	MovieID       int32     `json:"movie_id" db:"movie_id"`
	CinemaID      int32     `json:"cinema_id" db:"cinema_id"`
}
