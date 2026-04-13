package artifact_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/open-cli-collective/atlassian-go/artifact"
)

// This example demonstrates the artifact projection pattern that commands
// in #199 and #200 should follow. Real commands call v.RenderArtifact(),
// not json.Encoder directly — the direct encoding here is test scaffolding.
func Example_projection() {
	// 1. Define artifact struct with full-only fields using omitempty
	type IssueArtifact struct {
		Key     string `json:"key"`
		Summary string `json:"summary"`
		Status  string `json:"status"`
		// Full-only fields use omitempty
		Created string `json:"created,omitempty"`
	}

	// 2. Projection function: (domain data, mode) → artifact
	project := func(key, summary, status, created string, mode artifact.Type) *IssueArtifact {
		a := &IssueArtifact{Key: key, Summary: summary, Status: status}
		if mode.IsFull() {
			a.Created = created
		}
		return a
	}

	// 3. Output (test scaffolding - real code uses v.RenderArtifact())
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")

	// Agent mode: Created omitted
	_ = enc.Encode(project("PROJ-1", "Fix bug", "Open", "2024-01-01", artifact.Agent))
	fmt.Println("Agent mode:")
	fmt.Println(strings.TrimSpace(buf.String()))

	buf.Reset()

	// Full mode: Created included
	_ = enc.Encode(project("PROJ-1", "Fix bug", "Open", "2024-01-01", artifact.Full))
	fmt.Println("\nFull mode:")
	fmt.Println(strings.TrimSpace(buf.String()))

	// Output:
	// Agent mode:
	// {
	//   "key": "PROJ-1",
	//   "summary": "Fix bug",
	//   "status": "Open"
	// }
	//
	// Full mode:
	// {
	//   "key": "PROJ-1",
	//   "summary": "Fix bug",
	//   "status": "Open",
	//   "created": "2024-01-01"
	// }
}

// This example demonstrates how to use NewListResult to wrap a slice
// of artifacts with pagination metadata.
func Example_listResult() {
	type PageArtifact struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}

	pages := []*PageArtifact{
		{ID: "123", Title: "Getting Started"},
		{ID: "456", Title: "Advanced Topics"},
	}

	result := artifact.NewListResult(pages, true)

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	_ = enc.Encode(result)

	fmt.Println(strings.TrimSpace(buf.String()))

	// Output:
	// {
	//   "results": [
	//     {
	//       "id": "123",
	//       "title": "Getting Started"
	//     },
	//     {
	//       "id": "456",
	//       "title": "Advanced Topics"
	//     }
	//   ],
	//   "_meta": {
	//     "count": 2,
	//     "hasMore": true
	//   }
	// }
}
