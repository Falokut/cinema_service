package repository

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type GeoPoint struct {
	Latityde, Longitude float64
}

func (p *GeoPoint) Scan(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return errors.New("unsupported value type")
	}

	str = strings.Trim(str, "POINT()")
	nums := strings.Split(str, " ")
	if len(nums) != 2 {
		return errors.New("invalid data, can't parse point")
	}

	X, err := strconv.ParseFloat(nums[0], 64)
	if err != nil {
		return errors.New("invalid Latityde data, can't parse point,")
	}

	Y, err := strconv.ParseFloat(nums[1], 64)
	if err != nil {
		return errors.New("invalid Longitude data, can't parse point")
	}

	p.Latityde = X
	p.Longitude = Y
	return nil
}

func (p GeoPoint) Value() (driver.Value, error) {
	return fmt.Sprintf("POINT(%f %f)", p.Latityde, p.Longitude), nil
}
