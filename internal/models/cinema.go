package models

type Cinema struct {
	Name        string   `json:"name" db:"name"`
	Address     string   `json:"address" db:"address"`
	Coordinates GeoPoint `json:"coordinates" db:"coordinates"`
	ID          int32    `json:"id" db:"id"`
}
