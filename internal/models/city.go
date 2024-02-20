package models

type City struct {
	Name string `json:"name" db:"name"`
	ID   int32  `json:"id" db:"id"`
}
