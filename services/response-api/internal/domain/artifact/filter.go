package artifact

import "time"

// Filter contains criteria for filtering artifacts.
type Filter struct {
	ID          *string
	ResponseID  *string
	PlanID      *string
	ContentType *ContentType
	IsLatest    *bool

	// Retention filter
	RetentionPolicy *RetentionPolicy
	ExcludeExpired  bool

	// Time filters
	CreatedAfter  *time.Time
	CreatedBefore *time.Time

	// Pagination
	Limit  int
	Offset int
}

// NewFilter creates a new filter with default pagination.
func NewFilter() *Filter {
	return &Filter{
		Limit:          20,
		Offset:         0,
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
func (f *Filter) WithPagination(limit, offset int) *Filter {
	f.Limit = limit
	f.Offset = offset
	return f
}

// IncludeExpired includes expired artifacts in results.
func (f *Filter) IncludeExpired() *Filter {
	f.ExcludeExpired = false
	return f
}
