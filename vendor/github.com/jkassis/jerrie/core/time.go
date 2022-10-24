package core

import (
	"strconv"
	"time"
)

// Time is a time object that json serializes to milliseconds, not nanoseconds
type Time struct {
	time.Time
}

// MarshalJSON implements the json.Marshaler interface.
func (t *Time) MarshalJSON() ([]byte, error) {
	timeNs := t.UnixNano()
	timeMs := timeNs / (int64(time.Millisecond) / int64(time.Nanosecond))
	return []byte(strconv.FormatInt(timeMs, 10)), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (t *Time) UnmarshalJSON(data []byte) error {
	// Ignore null, like in the main JSON package.
	if string(data) == "null" {
		return nil
	}
	timeMs, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return err
	}
	timeNs := timeMs * (int64(time.Millisecond) / int64(time.Nanosecond))
	t.Time = time.Unix(0, timeNs)
	return nil
}

// Duration is a duraiton object that json serializes to milliseconds, not nanoseconds
type Duration struct {
	time.Duration
}

// MarshalJSON implements the json.Marshaler interface.
func (d *Duration) MarshalJSON() ([]byte, error) {
	durationMs := d.Duration.Truncate(time.Millisecond) / time.Millisecond
	return []byte(strconv.FormatInt(int64(durationMs), 10)), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (d *Duration) UnmarshalJSON(data []byte) error {
	// Ignore null, like in the main JSON package.
	if string(data) == "null" {
		return nil
	}
	durationMs, err := strconv.ParseInt(string(data[:]), 10, 64)
	if err != nil {
		return err
	}
	duration := durationMs * int64(time.Millisecond)
	d.Duration = time.Duration(duration)
	return nil
}

type Timer struct {
	start time.Time
	time.Duration
}

// Start sets the initial time for a timer
func (t *Timer) Start() {
	t.start = time.Now()
}

// End set the duration
func (t *Timer) End() {
	t.Duration = time.Now().Sub(t.start)
}

// String returns a friendly string
func (t *Timer) String() string {
	return t.Duration.String()
}
