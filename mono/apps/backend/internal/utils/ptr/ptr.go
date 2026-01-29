package ptr

import "time"

func ToString(s string) *string {
	return &s
}

func ToInt(i int) *int {
	return &i
}

func ToInt64(i int64) *int64 {
	return &i
}

func ToUint(i uint) *uint {
	return &i
}

func ToBool(b bool) *bool {
	return &b
}

func ToTime(b time.Time) *time.Time {
	return &b
}

func ToFloat64(f float64) *float64 {
	return &f
}
