package utils

// BoolPtr returns a pointer to the bool value
func BoolPtr(b bool) *bool {
	return &b
}

// BoolValue returns the value of a bool pointer or false if nil
func BoolValue(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// BoolValueWithDefault returns the value of a bool pointer or the default if nil
func BoolValueWithDefault(b *bool, def bool) bool {
	if b == nil {
		return def
	}
	return *b
}

// IntPtr returns a pointer to the int value
func IntPtr(i int) *int {
	return &i
}

// IntValue returns the value of an int pointer or 0 if nil
func IntValue(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

// IntValueWithDefault returns the value of an int pointer or the default if nil
func IntValueWithDefault(i *int, def int) int {
	if i == nil {
		return def
	}
	return *i
}

// Int64Ptr returns a pointer to the int64 value
func Int64Ptr(i int64) *int64 {
	return &i
}

// Int64Value returns the value of an int64 pointer or 0 if nil
func Int64Value(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

// Float64Ptr returns a pointer to the float64 value
func Float64Ptr(f float64) *float64 {
	return &f
}

// Float64Value returns the value of a float64 pointer or 0 if nil
func Float64Value(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}
