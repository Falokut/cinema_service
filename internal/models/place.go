package models

type Place struct {
	Row      int32   `json:"row" db:"row"`
	Seat     int32   `json:"seat" db:"seat"`
	GridPosX float32 `json:"grid_pos_x" db:"grid_pos_x"`
	GridPosY float32 `json:"grid_pos_y" db:"grid_pos_y"`
}
