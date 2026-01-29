package query

type Pagination struct {
	Limit  *int
	Offset *int
	After  *uint
	Order  string
}
