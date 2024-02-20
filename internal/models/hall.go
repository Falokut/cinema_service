package models

type Hall struct {
	Type string `db:"hall_type" json:"hall_type"`
	Name string `db:"name" json:"name"`
	Size uint32 `db:"size" json:"size"`
	Id   int32  `db:"id" json:"id"`
}
