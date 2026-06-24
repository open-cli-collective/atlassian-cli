package present

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"

	"github.com/open-cli-collective/atlassian-go/atime"
	sharedpresent "github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/confluence-cli/api"
)

type SpacePresenter struct{}
type PagePresenter struct{}
type PageHistoryPresenter struct{}
type SearchPresenter struct{}
type AttachmentPresenter struct{}

var spaceKeyRegex = regexp.MustCompile(`/spaces/([^/]+)`)

func (SpacePresenter) PresentList(spaces []api.Space, full bool, nextCursor string, hasMore bool) *sharedpresent.OutputModel {
	headers := []string{"ID", "KEY", "TYPE", "NAME"}
	if full {
		headers = []string{"ID", "KEY", "TYPE", "STATUS", "NAME"}
	}

	rows := make([]sharedpresent.Row, 0, len(spaces))
	for _, space := range spaces {
		cells := []string{orDash(space.ID), orDash(space.Key), orDash(space.Type), orDash(space.Name)}
		if full {
			cells = []string{orDash(space.ID), orDash(space.Key), orDash(space.Type), orDash(space.Status), orDash(space.Name)}
		}
		rows = append(rows, sharedpresent.Row{Cells: cells})
	}

	sections := []sharedpresent.Section{&sharedpresent.TableSection{Headers: headers, Rows: rows}}
	if nextCursor != "" {
		sections = append(sections, stderrInfo(fmt.Sprintf(`Next page: cfl space list --cursor %q`, nextCursor)))
	} else if hasMore {
		sections = append(sections, stderrInfo(fmt.Sprintf("(showing first %d results, use --limit to see more)", len(spaces))))
	}

	return &sharedpresent.OutputModel{Sections: sections}
}

func (SpacePresenter) PresentEmpty() *sharedpresent.OutputModel {
	return stderrOnly("No spaces found.")
}

func (PagePresenter) PresentList(pages []api.Page, full, hasMore bool) *sharedpresent.OutputModel {
	headers := []string{"ID", "TITLE", "STATUS"}
	if full {
		headers = []string{"ID", "TITLE", "STATUS", "VERSION", "PARENT ID"}
	}

	rows := make([]sharedpresent.Row, 0, len(pages))
	for _, page := range pages {
		cells := []string{orDash(page.ID), truncateText(page.Title, 60), orDash(page.Status)}
		if full {
			cells = append(cells, pageVersionCell(page.Version), orDash(page.ParentID))
		}
		rows = append(rows, sharedpresent.Row{Cells: cells})
	}

	sections := []sharedpresent.Section{&sharedpresent.TableSection{Headers: headers, Rows: rows}}
	if hasMore {
		sections = append(sections, stderrInfo(fmt.Sprintf("(showing first %d results, use --limit to see more)", len(pages))))
	}

	return &sharedpresent.OutputModel{Sections: sections}
}

func (PagePresenter) PresentEmpty(spaceKey string) *sharedpresent.OutputModel {
	if spaceKey == "" {
		return stderrOnly("No pages found.")
	}
	return stderrOnly(fmt.Sprintf("No pages found in space %s.", spaceKey))
}

func (PageHistoryPresenter) PresentList(versions []api.Version, nextCursor, pageID string, hasMore bool) *sharedpresent.OutputModel {
	rows := make([]sharedpresent.Row, 0, len(versions))
	for _, version := range versions {
		rows = append(rows, sharedpresent.Row{Cells: []string{
			strconv.Itoa(version.Number),
			formatHistoryTime(version.CreatedAt),
			orDash(version.AuthorID),
		}})
	}

	sections := []sharedpresent.Section{
		&sharedpresent.TableSection{
			Headers: []string{"VERSION", "CREATED", "AUTHOR"},
			Rows:    rows,
		},
	}
	if nextCursor != "" {
		sections = append(sections, stderrInfo(fmt.Sprintf(`Next page: cfl page history list %s --cursor %q`, pageID, nextCursor)))
	} else if hasMore {
		sections = append(sections, stderrInfo(fmt.Sprintf("(showing first %d results, use --limit to see more)", len(versions))))
	}

	return &sharedpresent.OutputModel{Sections: sections}
}

func (PageHistoryPresenter) PresentIDs(versions []api.Version, nextCursor, pageID string, hasMore bool) *sharedpresent.OutputModel {
	sections := make([]sharedpresent.Section, 0, len(versions)+1)
	for _, version := range versions {
		sections = append(sections, stdoutInfo(strconv.Itoa(version.Number)))
	}
	if nextCursor != "" {
		sections = append(sections, stderrInfo(fmt.Sprintf(`Next page: cfl page history list %s --cursor %q`, pageID, nextCursor)))
	} else if hasMore {
		sections = append(sections, stderrInfo(fmt.Sprintf("(showing first %d results, use --limit to see more)", len(versions))))
	}

	return &sharedpresent.OutputModel{Sections: sections}
}

func (PageHistoryPresenter) PresentEmpty() *sharedpresent.OutputModel {
	return stderrOnly("No page versions found.")
}

func (SearchPresenter) PresentList(results []api.SearchResult, full bool, totalSize int, hasMore bool) *sharedpresent.OutputModel {
	headers := []string{"ID", "TYPE", "SPACE", "TITLE"}
	if full {
		headers = []string{"ID", "TYPE", "SPACE", "TITLE", "MODIFIED", "URL"}
	}

	rows := make([]sharedpresent.Row, 0, len(results))
	for _, result := range results {
		cells := []string{
			orDash(result.Content.ID),
			orDash(result.Content.Type),
			orDash(ExtractSpaceKey(result.ResultGlobalContainer.DisplayURL)),
			truncateText(result.Content.Title, 50),
		}
		if full {
			cells = append(cells, orDash(result.LastModified), orDash(result.URL))
		}
		rows = append(rows, sharedpresent.Row{Cells: cells})
	}

	sections := []sharedpresent.Section{&sharedpresent.TableSection{Headers: headers, Rows: rows}}
	if hasMore {
		sections = append(sections, stderrInfo(fmt.Sprintf("(showing %d of %d results, use --limit to see more)", len(results), totalSize)))
	}

	return &sharedpresent.OutputModel{Sections: sections}
}

func (SearchPresenter) PresentEmpty() *sharedpresent.OutputModel {
	return stderrOnly("No results found.")
}

func (AttachmentPresenter) PresentList(attachments []api.Attachment, full, hasMore bool) *sharedpresent.OutputModel {
	headers := []string{"ID", "TITLE", "MEDIA TYPE", "FILE SIZE"}
	if full {
		headers = []string{"ID", "TITLE", "MEDIA TYPE", "FILE SIZE", "STATUS", "COMMENT"}
	}

	rows := make([]sharedpresent.Row, 0, len(attachments))
	for _, attachment := range attachments {
		cells := []string{
			orDash(attachment.ID),
			orDash(attachment.Title),
			orDash(attachment.MediaType),
			formatAttachmentFileSize(attachment.FileSize),
		}
		if full {
			cells = append(cells, orDash(attachment.Status), orDash(attachment.Comment))
		}
		rows = append(rows, sharedpresent.Row{Cells: cells})
	}

	sections := []sharedpresent.Section{&sharedpresent.TableSection{Headers: headers, Rows: rows}}
	if hasMore {
		sections = append(sections, stderrInfo(fmt.Sprintf("(showing first %d results, use --limit to see more)", len(attachments))))
	}

	return &sharedpresent.OutputModel{Sections: sections}
}

func (AttachmentPresenter) PresentEmpty(unused bool) *sharedpresent.OutputModel {
	if unused {
		return stderrOnly("No unused attachments found.")
	}
	return stderrOnly("No attachments found.")
}

func stderrOnly(message string) *sharedpresent.OutputModel {
	return &sharedpresent.OutputModel{Sections: []sharedpresent.Section{stderrInfo(message)}}
}

func stderrInfo(message string) *sharedpresent.MessageSection {
	return &sharedpresent.MessageSection{
		Kind:    sharedpresent.MessageInfo,
		Message: message,
		Stream:  sharedpresent.StreamStderr,
	}
}

func stdoutInfo(message string) *sharedpresent.MessageSection {
	return &sharedpresent.MessageSection{
		Kind:    sharedpresent.MessageInfo,
		Message: message,
		Stream:  sharedpresent.StreamStdout,
	}
}

func orDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func truncateText(value string, max int) string {
	if value == "" {
		return "-"
	}
	if max <= 0 || len(value) <= max {
		return value
	}
	if max <= 3 {
		return value[:max]
	}
	return value[:max-3] + "..."
}

func pageVersionCell(version *api.Version) string {
	if version == nil {
		return "-"
	}
	return fmt.Sprintf("v%d", version.Number)
}

func formatHistoryTime(t *atime.AtlassianTime) string {
	if t == nil || t.IsZero() {
		return "-"
	}
	return t.UTC().Format("2006-01-02T15:04:05Z07:00")
}

func ExtractSpaceKey(displayURL string) string {
	matches := spaceKeyRegex.FindStringSubmatch(displayURL)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func ExtractCursor(nextLink string) string {
	if nextLink == "" {
		return ""
	}
	parsed, err := url.Parse(nextLink)
	if err != nil {
		return ""
	}
	return parsed.Query().Get("cursor")
}

func formatAttachmentFileSize(bytes int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)

	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.1f GB", float64(bytes)/gb)
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/mb)
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/kb)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
