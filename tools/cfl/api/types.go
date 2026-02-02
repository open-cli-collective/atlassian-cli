// Package api provides the Confluence Cloud REST API client.
package api

import "time"

// PaginatedResponse wraps paginated API responses.
type PaginatedResponse[T any] struct {
	Results []T   `json:"results"`
	Links   Links `json:"_links,omitempty"`
}

// Links contains pagination and navigation links.
type Links struct {
	Next   string `json:"next,omitempty"`
	Base   string `json:"base,omitempty"`
	WebUI  string `json:"webui,omitempty"`
	EditUI string `json:"editui,omitempty"`
}

// HasMore returns true if there are more results available.
func (p *PaginatedResponse[T]) HasMore() bool {
	return p.Links.Next != ""
}

// Space represents a Confluence space.
type Space struct {
	ID          string            `json:"id"`
	Key         string            `json:"key"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Status      string            `json:"status"`
	Description *SpaceDescription `json:"description,omitempty"`
	Links       Links             `json:"_links,omitempty"`
}

// SpaceDescription contains space description in various formats.
type SpaceDescription struct {
	Plain *DescriptionValue `json:"plain,omitempty"`
	View  *DescriptionValue `json:"view,omitempty"`
}

// DescriptionValue holds the actual description text.
type DescriptionValue struct {
	Value string `json:"value"`
}

// Page represents a Confluence page.
type Page struct {
	ID         string   `json:"id"`
	Status     string   `json:"status"`
	Title      string   `json:"title"`
	SpaceID    string   `json:"spaceId"`
	ParentID   string   `json:"parentId,omitempty"`
	ParentType string   `json:"parentType,omitempty"`
	Position   int      `json:"position,omitempty"`
	AuthorID   string   `json:"authorId,omitempty"`
	OwnerID    string   `json:"ownerId,omitempty"`
	CreatedAt  Time     `json:"createdAt,omitempty"`
	Version    *Version `json:"version,omitempty"`
	Body       *Body    `json:"body,omitempty"`
	Links      Links    `json:"_links,omitempty"`
}

// Version contains page version information.
type Version struct {
	Number    int    `json:"number"`
	Message   string `json:"message,omitempty"`
	MinorEdit bool   `json:"minorEdit,omitempty"`
	AuthorID  string `json:"authorId,omitempty"`
	CreatedAt Time   `json:"createdAt,omitempty"`
}

// Body contains page content in various representations.
type Body struct {
	Storage        *BodyRepresentation `json:"storage,omitempty"`
	AtlasDocFormat *BodyRepresentation `json:"atlas_doc_format,omitempty"`
	View           *BodyRepresentation `json:"view,omitempty"`
}

// BodyRepresentation holds content in a specific format.
type BodyRepresentation struct {
	Representation string `json:"representation"`
	Value          string `json:"value"`
}

// Attachment represents a file attachment.
type Attachment struct {
	ID                   string   `json:"id"`
	Status               string   `json:"status"`
	Title                string   `json:"title"`
	MediaType            string   `json:"mediaType"`
	MediaTypeDescription string   `json:"mediaTypeDescription,omitempty"`
	Comment              string   `json:"comment,omitempty"`
	FileSize             int64    `json:"fileSize"`
	WebuiLink            string   `json:"webuiLink,omitempty"`
	DownloadLink         string   `json:"downloadLink,omitempty"`
	Version              *Version `json:"version,omitempty"`
	Links                Links    `json:"_links,omitempty"`
}

// Time is a wrapper around time.Time for custom JSON parsing.
type Time struct {
	time.Time
}

// UnmarshalJSON parses Confluence's ISO 8601 date format.
func (t *Time) UnmarshalJSON(data []byte) error {
	s := string(data)

	// Handle null or empty
	if s == "null" || s == `""` || s == "" {
		return nil
	}

	// Remove quotes if present
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}

	if s == "" {
		return nil
	}

	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		// Try alternative format
		parsed, err = time.Parse("2006-01-02T15:04:05.000Z", s)
		if err != nil {
			return err
		}
	}

	t.Time = parsed
	return nil
}

// MarshalJSON formats time in ISO 8601 format.
func (t Time) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	return []byte(`"` + t.Format(time.RFC3339) + `"`), nil
}

// CreatePageRequest is the request body for creating a page.
type CreatePageRequest struct {
	SpaceID  string `json:"spaceId"`
	Status   string `json:"status,omitempty"`
	Title    string `json:"title"`
	ParentID string `json:"parentId,omitempty"`
	Body     *Body  `json:"body"`
}

// UpdatePageRequest is the request body for updating a page.
type UpdatePageRequest struct {
	ID      string   `json:"id"`
	Status  string   `json:"status"`
	Title   string   `json:"title"`
	Body    *Body    `json:"body"`
	Version *Version `json:"version"`
}

// ErrorResponse represents an API error.
type ErrorResponse struct {
	StatusCode int      `json:"statusCode"`
	Message    string   `json:"message"`
	Errors     []string `json:"errors,omitempty"`
}

func (e *ErrorResponse) Error() string {
	if len(e.Errors) > 0 {
		return e.Errors[0]
	}
	return e.Message
}

// User represents a Confluence user.
type User struct {
	AccountID   string `json:"accountId"`
	AccountType string `json:"accountType,omitempty"`
	Email       string `json:"email,omitempty"`
	PublicName  string `json:"publicName,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
}
