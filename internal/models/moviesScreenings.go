package models

type MoviesScreenings struct {
	ScreeningsTypes []string `json:"screenings_types" db:"screenings_types"`
	HallsTypes      []string `json:"halls_types" db:"halls_types"`
	MovieId         int32    `json:"movie_id" db:"movie_id"`
}
