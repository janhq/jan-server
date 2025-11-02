package query

// Pagination defines pagination parameters for queries
type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// NewPagination creates a new Pagination with defaults
func NewPagination(limit, offset int) *Pagination {
	if limit <= 0 {
		limit = 20 // default limit
	}
	if limit > 1000 {
		limit = 1000 // max limit
	}
	if offset < 0 {
		offset = 0
	}
	return &Pagination{
		Limit:  limit,
		Offset: offset,
	}
}
