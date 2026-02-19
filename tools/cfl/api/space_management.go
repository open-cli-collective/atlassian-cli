package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"fmt"
)

// CreateSpaceRequest is the request body for creating a space (v2 API).
type CreateSpaceRequest struct {
	Key         string            `json:"key"`
	Name        string            `json:"name"`
	Description *SpaceDescription `json:"description,omitempty"`
	Type        string            `json:"type,omitempty"`
}

// UpdateSpaceRequest is the request body for updating a space (v1 API).
type UpdateSpaceRequest struct {
	Key         string              `json:"key"`
	Name        string              `json:"name,omitempty"`
	Description *V1SpaceDescription `json:"description,omitempty"`
}

// V1SpaceDescription is the v1 API description format (includes representation field).
type V1SpaceDescription struct {
	Plain *V1DescriptionValue `json:"plain,omitempty"`
}

// V1DescriptionValue holds a description value with its representation type.
type V1DescriptionValue struct {
	Value          string `json:"value"`
	Representation string `json:"representation"`
}

// v1SpaceResponse is the v1 API response for space operations.
type v1SpaceResponse struct {
	ID          int    `json:"id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description struct {
		Plain struct {
			Value          string `json:"value"`
			Representation string `json:"representation"`
		} `json:"plain"`
	} `json:"description"`
	Links struct {
		WebUI string `json:"webui"`
		Self  string `json:"self"`
	} `json:"_links"`
}

// toSpace converts a v1 API response to a Space.
func (r *v1SpaceResponse) toSpace() *Space {
	space := &Space{
		ID:   fmt.Sprintf("%d", r.ID),
		Key:  r.Key,
		Name: r.Name,
		Type: r.Type,
		Links: Links{
			WebUI: r.Links.WebUI,
		},
	}
	if r.Description.Plain.Value != "" {
		space.Description = &SpaceDescription{
			Plain: &DescriptionValue{
				Value: r.Description.Plain.Value,
			},
		}
	}
	return space
}

// CreateSpace creates a new Confluence space.
func (c *Client) CreateSpace(ctx context.Context, req *CreateSpaceRequest) (*Space, error) {
	body, err := c.Post(ctx, "/api/v2/spaces", req)
	if err != nil {
		return nil, fmt.Errorf("creating space: %w", err)
	}

	var space Space
	if err := json.Unmarshal(body, &space); err != nil {
		return nil, fmt.Errorf("parsing create space response: %w", err)
	}

	return &space, nil
}

// UpdateSpace updates an existing Confluence space.
// Uses the v1 REST API as v2 doesn't support space updates.
func (c *Client) UpdateSpace(ctx context.Context, spaceKey string, req *UpdateSpaceRequest) (*Space, error) {
	path := fmt.Sprintf("/rest/api/space/%s", spaceKey)
	body, err := c.Put(ctx, path, req)
	if err != nil {
		return nil, fmt.Errorf("updating space: %w", err)
	}

	var response v1SpaceResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parsing update space response: %w", err)
	}

	return response.toSpace(), nil
}

// DeleteSpace deletes a Confluence space.
// Uses the v1 REST API as v2 doesn't support space deletion.
// The API returns 202 Accepted as deletion is asynchronous.
func (c *Client) DeleteSpace(ctx context.Context, spaceKey string) error {
	path := fmt.Sprintf("/rest/api/space/%s", spaceKey)
	_, err := c.Delete(ctx, path)
	if err != nil {
		return fmt.Errorf("deleting space %s: %w", spaceKey, err)
	}
	return nil
}
