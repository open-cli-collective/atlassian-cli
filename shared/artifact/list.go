package artifact

// ListResult wraps a slice of artifacts with pagination metadata.
// Use this for all list command outputs to ensure consistent structure.
type ListResult struct {
	Results any      `json:"results"`
	Meta    ListMeta `json:"_meta,omitempty"`
}

// ListMeta contains pagination metadata.
type ListMeta struct {
	Count   int  `json:"count"`
	HasMore bool `json:"hasMore,omitempty"`
}

// NewListResult creates a ListResult from a slice of artifacts.
func NewListResult[T any](items []T, hasMore bool) *ListResult {
	return &ListResult{
		Results: items,
		Meta: ListMeta{
			Count:   len(items),
			HasMore: hasMore,
		},
	}
}
