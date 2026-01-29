package artifact

import "time"

// Filter contains criteria for filtering artifacts.
type Filter struct {
	ID          *string
	ResponseID  *string
	PlanID      *string
	UserID      *string // Filter by user (via response.user_id join)
	ContentType *ContentType
	IsLatest    *bool
	TitleSearch *string // Search by title (ILIKE)

	// Retention filter
	RetentionPolicy *RetentionPolicy
	ExcludeExpired  bool

	// Time filters
	CreatedAfter  *time.Time
	CreatedBefore *time.Time

	// Pagination (cursor-based)
	Limit int
	After *uint  // Internal ID for cursor-based pagination
	Order string // "asc" or "desc" (default: "desc")
}

// NewFilter creates a new filter with default pagination.
func NewFilter() *Filter {
	return &Filter{
		Limit:          20,
		Order:          "desc",
		ExcludeExpired: true,
	}
}

// WithResponseID sets the response ID filter.
func (f *Filter) WithResponseID(responseID string) *Filter {
	f.ResponseID = &responseID
	return f
}

// WithPlanID sets the plan ID filter.
func (f *Filter) WithPlanID(planID string) *Filter {
	f.PlanID = &planID
	return f
}

// WithUserID sets the user ID filter (filters via response.user_id join).
func (f *Filter) WithUserID(userID string) *Filter {
	f.UserID = &userID
	return f
}

// WithTitleSearch sets the title search filter (case-insensitive).
func (f *Filter) WithTitleSearch(search string) *Filter {
	f.TitleSearch = &search
	return f
}

// WithContentType sets the content type filter.
func (f *Filter) WithContentType(contentType ContentType) *Filter {
	f.ContentType = &contentType
	return f
}

// WithLatestOnly filters to only latest versions.
func (f *Filter) WithLatestOnly() *Filter {
	isLatest := true
	f.IsLatest = &isLatest
	return f
}

// WithAllVersions includes all versions, not just latest.
func (f *Filter) WithAllVersions() *Filter {
	f.IsLatest = nil
	return f
}

// WithPagination sets the pagination parameters.
func (f *Filter) WithPagination(limit int, after *uint) *Filter {
	f.Limit = limit
	f.After = after
	return f
}

// WithAfter sets the cursor for pagination.
func (f *Filter) WithAfter(afterID uint) *Filter {
	f.After = &afterID
	return f
}

// WithOrder sets the sort order ("asc" or "desc").
func (f *Filter) WithOrder(order string) *Filter {
	if order == "asc" || order == "desc" {
		f.Order = order
	}
	return f
}

// IncludeExpired includes expired artifacts in results.
func (f *Filter) IncludeExpired() *Filter {
	f.ExcludeExpired = false
	return f
}
