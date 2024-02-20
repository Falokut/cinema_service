package models

import "time"

type CityScreening struct {
	ScreeningType string    `json:"screening_type" db:"screening_type"`
	TicketPrice   string    `json:"ticket_price" db:"ticket_price"`
	StartTime     time.Time `json:"start_time" db:"start_time"`
	ScreeningId   int64     `json:"id" db:"id"`
	HallId        int32     `json:"hall_id" db:"hall_id"`
	CinemaId      int32     `json:"cinema_id" db:"cinema_id"`
}
