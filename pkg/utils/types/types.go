package types

import (
	"errors"
	"strings"
	"time"
)

const (
	YYYYMMDD string = "2006-01-02"
)

type Date time.Time

func (d *Date) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" {
		*d = Date(time.Time{})
		return errors.New("empty or invalid date field")
	}
	t, err := time.Parse(YYYYMMDD, s)
	if err != nil {
		return err
	}
	*d = Date(t)
	return nil
}

func (d Date) Time() time.Time {
	return time.Time(d)
}
