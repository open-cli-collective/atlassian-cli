package issues

import (
	"context"
	"errors"
	"time"

	sharederrors "github.com/open-cli-collective/atlassian-go/errors"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

var errStatusUnavailable = errors.New("task status unavailable after retries")

var errTypeChangeUnverified = errors.New("type change accepted but status could not be verified")

var pollRetryDelay = 2 * time.Second

const maxPollNotFoundRetries = 3

// pollMoveTask polls GetMoveTaskStatus until a terminal state is reached.
// Transient 404s are retried up to maxPollNotFoundRetries times.
// Persistent 404s return errStatusUnavailable.
func pollMoveTask(ctx context.Context, client *api.Client, taskID string) (*api.MoveTaskStatus, error) {
	notFoundRetries := 0

	for {
		status, err := client.GetMoveTaskStatus(ctx, taskID)
		if err != nil {
			if sharederrors.IsNotFound(err) && notFoundRetries < maxPollNotFoundRetries {
				notFoundRetries++
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(pollRetryDelay):
				}
				continue
			}
			if sharederrors.IsNotFound(err) {
				return nil, errStatusUnavailable
			}
			return nil, err
		}
		notFoundRetries = 0

		switch status.Status {
		case "COMPLETE", "FAILED", "CANCELLED":
			return status, nil
		case "ENQUEUED", "RUNNING":
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(1 * time.Second):
			}
		default:
			return status, nil
		}
	}
}
