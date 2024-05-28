package data

import (
	"strings"

	"github.com/rynhndrcksn/greenlight/internal/validator"
)

type Filters struct {
	Page         int
	PageSize     int
	Sort         string
	SortSafeList []string
}

// ValidateFilters validates all the filters being passed to the API.
func ValidateFilters(v *validator.Validator, f Filters) {
	// Check that the page and page_size parameters contain sensible values.
	v.Check(f.Page > 0, "page", "must be greater than zero")
	v.Check(f.Page <= 10_000_000, "page", "must be a maximum of 10 million")
	v.Check(f.PageSize > 0, "page_size", "must be greater than zero")
	v.Check(f.PageSize <= 100, "page_size", "must be a maximum of 100")

	// Check that the sort parameter matches a value in the safe list.
	v.Check(validator.PermittedValue(f.Sort, f.SortSafeList...), "sort", "invalid sort value")
}

// sortColumn determines which column, if any, we're sorting by.
func (f Filters) sortColumn() string {
	for _, safeValue := range f.SortSafeList {
		if f.Sort == safeValue {
			return strings.TrimPrefix(f.Sort, "-")
		}
	}

	// In theory, this should never be reached as ValidateFilters()
	// should take care of this long before this gets called.
	panic("unsafe sort parameter: " + f.Sort)
}

// sortDirection determines whether we're sorting by "ASC" or "DESC".
func (f Filters) sortDirection() string {
	if strings.HasPrefix(f.Sort, "-") {
		return "DESC"
	}

	return "ASC"
}

// limit determines pagination limit.
func (f Filters) limit() int {
	return f.PageSize
}

// offset determines pagination offset.
func (f Filters) offset() int {
	// Note: technically, we run the risk of an integer overflow by multiplying two integers.
	// However, our validation rules (page_size <= 100 and page <= 10_000_000) prevent this.
	return (f.Page - 1) * f.PageSize
}
