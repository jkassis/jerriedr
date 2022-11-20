package schema

import "time"

type TimeFilter struct {
	Year    *int
	Month   *time.Month
	Day     *int
	Weekday *time.Weekday
	Hour    *int
	Minute  *int
	Second  *int
}

func (tf *TimeFilter) isOK(t time.Time) bool {
	if tf.Year != nil && *tf.Year != t.Year() {
		return false
	}
	if tf.Month != nil && *tf.Month != t.Month() {
		return false
	}
	if tf.Day != nil && *tf.Day != t.Day() {
		return false
	}
	if tf.Weekday != nil && *tf.Weekday != t.Weekday() {
		return false
	}
	if tf.Hour != nil && *tf.Hour != t.Hour() {
		return false
	}
	if tf.Minute != nil && *tf.Minute != t.Minute() {
		return false
	}
	if tf.Second != nil && *tf.Second != t.Second() {
		return false
	}

	return true
}
