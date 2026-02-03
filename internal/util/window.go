package util

import (
	"fmt"
	"time"
)

// InWindow returns true if now is within the configured window.
// Empty window values mean no restriction.
func InWindow(now time.Time, start, end, tz string) (bool, error) {
	if start == "" && end == "" {
		return true, nil
	}
	loc := now.Location()
	if tz != "" {
		var err error
		loc, err = time.LoadLocation(tz)
		if err != nil {
			return false, fmt.Errorf("invalid timezone: %w", err)
		}
	}
	parse := func(v string) (time.Time, error) {
		if v == "" {
			return time.Time{}, nil
		}
		parsed, err := time.ParseInLocation("15:04", v, loc)
		if err != nil {
			return time.Time{}, err
		}
		return parsed, nil
	}
	startTime, err := parse(start)
	if err != nil {
		return false, fmt.Errorf("invalid window start: %w", err)
	}
	endTime, err := parse(end)
	if err != nil {
		return false, fmt.Errorf("invalid window end: %w", err)
	}
	current := time.Date(now.In(loc).Year(), now.In(loc).Month(), now.In(loc).Day(), now.In(loc).Hour(), now.In(loc).Minute(), 0, 0, loc)

	if start != "" && end == "" {
		startToday := time.Date(current.Year(), current.Month(), current.Day(), startTime.Hour(), startTime.Minute(), 0, 0, loc)
		return !current.Before(startToday), nil
	}
	if start == "" && end != "" {
		endToday := time.Date(current.Year(), current.Month(), current.Day(), endTime.Hour(), endTime.Minute(), 0, 0, loc)
		return !current.After(endToday), nil
	}
	startToday := time.Date(current.Year(), current.Month(), current.Day(), startTime.Hour(), startTime.Minute(), 0, 0, loc)
	endToday := time.Date(current.Year(), current.Month(), current.Day(), endTime.Hour(), endTime.Minute(), 0, 0, loc)

	if endToday.After(startToday) {
		return !current.Before(startToday) && !current.After(endToday), nil
	}
	// Window wraps past midnight.
	return !current.Before(startToday) || !current.After(endToday), nil
}
