package utils

import (
	"strconv"
)

const (
	DefaultLimit  = 20
	DefaultOffset = 0
	MaxLimit      = 100
)

// Pagination holds pagination parameters
type Pagination struct {
	Limit  int
	Offset int
}

// NewPagination creates a new Pagination with validated values
func NewPagination(limit, offset int) Pagination {
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	if offset < 0 {
		offset = DefaultOffset
	}
	return Pagination{
		Limit:  limit,
		Offset: offset,
	}
}

// ParsePagination parses pagination from string parameters
func ParsePagination(limitStr, offsetStr string) Pagination {
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = DefaultLimit
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = DefaultOffset
	}
	return NewPagination(limit, offset)
}
