package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type Date struct {
	Day   int
	Month int
	Year  int
}

func NewDate(day int, month int, year int) (Date, error) {
	return ParseDate(fmt.Sprintf(`%04d-%02d-%02d`, year, month, day))
}

func (d Date) String() string {
	return fmt.Sprintf(`%04d-%02d-%02d`, d.Year, d.Month, d.Day)
}

func (d Date) ToTime() time.Time {
	parsed, _ := time.Parse(time.DateOnly, d.String())

	return parsed
}

func ParseDate(t string) (Date, error) {
	parsed, err := time.Parse(time.DateOnly, t)

	if err != nil {
		return Date{}, fmt.Errorf("formato de fecha inválido")
	}
	y, m, d := parsed.Date()

	return Date{
		Day:   d,
		Month: int(m),
		Year:  y,
	}, nil
}

func DateOf(t time.Time) Date {
	year, month, day := t.Date()
	result, _ := NewDate(day, int(month), year)
	return result
}

func (d *Date) UnmarshalText(b []byte) error {
	parsed, err := ParseDate(string(b))

	if err != nil {
		return err
	}

	*d = parsed
	return nil
}

func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d Date) Value() (driver.Value, error) {
	return d.ToTime(), nil
}

func (d *Date) Scan(src any) error {
	t, ok := src.(time.Time)

	if !ok {
		return fmt.Errorf("domain.Date: no se puede escanear %T", src)
	}

	*d = DateOf(t)
	return nil
}

func (d Date) IsZero() bool {
	return d.Day == 0 && d.Month == 0 && d.Year == 0
}
