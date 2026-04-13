package artifact

// ListResult wraps a slice of artifacts with pagination metadata.
// Use this for all list command outputs to ensure consistent structure.
//
// Always use NewListResult to construct instances — it normalizes nil slices
// to empty slices to ensure proper JSON serialization.
type ListResult struct {
	Results any      `json:"results"`
	Meta    ListMeta `json:"_meta"`
}

// ListMeta contains pagination metadata.
type ListMeta struct {
	Count   int  `json:"count"`
	HasMore bool `json:"hasMore"`
}

// NewListResult creates a ListResult from a slice of artifacts.
// A nil slice is normalized to an empty slice to ensure JSON serializes
// as [] rather than null.
func NewListResult[T any](items []T, hasMore bool) *ListResult {
	if items == nil {
		items = []T{}
	}
	return &ListResult{
		Results: items,
		Meta: ListMeta{
			Count:   len(items),
			HasMore: hasMore,
		},
	}
}
