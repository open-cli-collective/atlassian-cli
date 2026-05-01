package cache

import (
	"context"
	"strings"
	"time"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// GetBoardsCacheFirst returns a BoardsResponse from the boards cache when
// fresh, applying project filter and pagination client-side. Falls back to
// client.ListBoards on stale/miss/error.
//
// The boards cache stores []api.Board (all boards, exhaustively paginated by
// fetchBoards). Client-side filtering and slicing reproduce the server-side
// pagination contract.
func GetBoardsCacheFirst(ctx context.Context, client *api.Client, projectFilter string, startAt, maxResults int) (*api.BoardsResponse, error) {
	entry, err := Lookup("boards")
	if err != nil {
		return client.ListBoards(ctx, projectFilter, startAt, maxResults)
	}

	env, err := ReadResource[[]api.Board]("boards")
	if err != nil {
		return client.ListBoards(ctx, projectFilter, startAt, maxResults)
	}

	switch Classify(env.FetchedAt, entry.TTL, time.Now()) {
	case StatusFresh, StatusManual:
		return boardsFromCache(env.Data, projectFilter, startAt, maxResults), nil
	case StatusStale, StatusUninitialized:
		return client.ListBoards(ctx, projectFilter, startAt, maxResults)
	case StatusUnavailable:
		return client.ListBoards(ctx, projectFilter, startAt, maxResults)
	}
	return client.ListBoards(ctx, projectFilter, startAt, maxResults)
}

func boardsFromCache(boards []api.Board, projectFilter string, startAt, maxResults int) *api.BoardsResponse {
	var filtered []api.Board
	if projectFilter != "" {
		for _, b := range boards {
			if strings.EqualFold(b.Location.ProjectKey, projectFilter) {
				filtered = append(filtered, b)
			}
		}
	} else {
		filtered = boards
	}

	if maxResults <= 0 {
		maxResults = 50
	}

	total := len(filtered)
	if startAt > total {
		startAt = total
	}
	end := startAt + maxResults
	if end > total {
		end = total
	}

	return &api.BoardsResponse{
		MaxResults: maxResults,
		StartAt:    startAt,
		Total:      total,
		IsLast:     end >= total,
		Values:     filtered[startAt:end],
	}
}
