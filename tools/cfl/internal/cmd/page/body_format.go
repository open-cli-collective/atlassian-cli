package page

import (
	"encoding/json"
	"fmt"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/pageview"
	"github.com/open-cli-collective/confluence-cli/pkg/md"
)

const (
	bodyFormatMarkdown = pageview.BodyFormatMarkdown
	bodyFormatADF      = pageview.BodyFormatADF
	bodyFormatXHTML    = pageview.BodyFormatXHTML
)

func resolveBodyFormat(value string, explicit bool) (string, error) {
	if value == "" && !explicit {
		return bodyFormatMarkdown, nil
	}
	switch value {
	case bodyFormatMarkdown, bodyFormatADF, bodyFormatXHTML:
		return value, nil
	default:
		return "", fmt.Errorf("invalid body format %q: must be markdown, adf, or xhtml", value)
	}
}

func bodyForInput(content, format string, legacy bool) (*api.Body, error) {
	if legacy && format != bodyFormatMarkdown {
		return nil, fmt.Errorf("--legacy is only supported with --body-format markdown")
	}

	switch format {
	case bodyFormatMarkdown:
		if legacy {
			converted, err := md.ToConfluenceStorage([]byte(content))
			if err != nil {
				return nil, fmt.Errorf("converting markdown: %w", err)
			}
			return storageBody(converted), nil
		}
		converted, err := md.ToADF([]byte(content))
		if err != nil {
			return nil, fmt.Errorf("converting markdown to ADF: %w", err)
		}
		return adfBody(converted), nil
	case bodyFormatADF:
		if !json.Valid([]byte(content)) {
			return nil, fmt.Errorf("invalid ADF JSON")
		}
		return adfBody(content), nil
	case bodyFormatXHTML:
		return storageBody(content), nil
	default:
		return nil, fmt.Errorf("unsupported body format %q", format)
	}
}

func adfBody(content string) *api.Body {
	return &api.Body{AtlasDocFormat: &api.BodyRepresentation{
		Representation: "atlas_doc_format",
		Value:          content,
	}}
}

func storageBody(content string) *api.Body {
	return &api.Body{Storage: &api.BodyRepresentation{
		Representation: "storage",
		Value:          content,
	}}
}
