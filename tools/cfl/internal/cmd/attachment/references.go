package attachment

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/pageview"
)

// attachmentRefs is the set of this page's attachments that its body
// references, keyed by filename and by media file id.
type attachmentRefs struct {
	filenames map[string]bool
	fileIDs   map[string]bool
}

func newAttachmentRefs() attachmentRefs {
	return attachmentRefs{filenames: map[string]bool{}, fileIDs: map[string]bool{}}
}

func (r attachmentRefs) references(att api.Attachment) bool {
	if att.FileID != "" && r.fileIDs[att.FileID] {
		return true
	}
	return r.filenames[att.Title]
}

// pageAttachmentRefs collects the attachment references in whichever body
// representation the page provides.
func pageAttachmentRefs(page *api.Page) (attachmentRefs, error) {
	switch {
	case pageview.HasStorageContent(page):
		return storageAttachmentRefs(page.Body.Storage.Value, page.ID, page.Title)
	case pageview.HasADFContent(page):
		return adfAttachmentRefs(page.Body.AtlasDocFormat.Value, page.ID)
	default:
		// Even a blank page has an ADF document, so no body at all means the
		// content could not be read. Reporting every attachment as unused
		// would invite deleting attachments the page still shows.
		return attachmentRefs{}, errors.New("page returned no storage or ADF body; cannot determine attachment usage")
	}
}

// storageAttachmentRefs collects ri:attachment elements and links to this
// page's attachment download URLs from storage XHTML. An ri:attachment that names another page or blog post points at that
// content's attachment, not this page's, so it is not collected. One naming a
// page with this page's title is kept: when unsure, reporting an attachment
// as used is the safe error.
func storageAttachmentRefs(storage, pageID, pageTitle string) (attachmentRefs, error) {
	refs := newAttachmentRefs()
	dec := xml.NewDecoder(strings.NewReader(storage))
	dec.Strict = false
	dec.Entity = xml.HTMLEntity

	var (
		inAttachment bool
		filename     string
		foreign      bool
	)
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return refs, nil
		}
		if err != nil {
			return attachmentRefs{}, fmt.Errorf("parsing page storage body: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Space == "" && t.Name.Local == "a" {
				for _, attr := range t.Attr {
					if attr.Name.Local == "href" {
						refs.addDownloadLink(attr.Value, pageID)
					}
				}
				continue
			}
			if t.Name.Space != "ri" {
				continue
			}
			if t.Name.Local == "attachment" {
				inAttachment, foreign, filename = true, false, ""
				for _, attr := range t.Attr {
					if attr.Name.Space == "ri" && attr.Name.Local == "filename" {
						filename = attr.Value
					}
				}
				continue
			}
			if inAttachment && !namesPage(t, pageTitle) {
				foreign = true
			}
		case xml.EndElement:
			if t.Name.Space == "ri" && t.Name.Local == "attachment" && inAttachment {
				if filename != "" && !foreign {
					refs.filenames[filename] = true
				}
				inAttachment = false
			}
		}
	}
}

func namesPage(el xml.StartElement, title string) bool {
	if el.Name.Local != "page" || title == "" {
		return false
	}
	for _, attr := range el.Attr {
		if attr.Name.Space == "ri" && attr.Name.Local == "content-title" {
			return attr.Value == title
		}
	}
	return false
}

// addDownloadLink records the attachment a link points at when the link is
// this page's attachment download URL.
func (r attachmentRefs) addDownloadLink(href, pageID string) {
	u, err := url.Parse(href)
	if err != nil {
		return
	}
	prefix := "/download/attachments/" + pageID + "/"
	i := strings.Index(u.Path, prefix)
	if i < 0 {
		return
	}
	if name := u.Path[i+len(prefix):]; name != "" && !strings.Contains(name, "/") {
		r.filenames[name] = true
	}
}

type adfNode struct {
	Type    string         `json:"type"`
	Attrs   map[string]any `json:"attrs"`
	Marks   []adfNode      `json:"marks"`
	Content []adfNode      `json:"content"`
}

// adfAttachmentRefs collects media and mediaInline file nodes, and links to
// this page's attachment download URLs, from an ADF document. Media nodes identify the file by its media file id; the filename
// attributes are only trusted when the node belongs to this page's collection.
func adfAttachmentRefs(value, pageID string) (attachmentRefs, error) {
	var doc adfNode
	if err := json.Unmarshal([]byte(value), &doc); err != nil {
		return attachmentRefs{}, fmt.Errorf("parsing page ADF body: %w", err)
	}
	refs := newAttachmentRefs()
	ownCollection := "contentId-" + pageID
	var walk func(n adfNode)
	walk = func(n adfNode) {
		if (n.Type == "media" || n.Type == "mediaInline") && attrString(n.Attrs, "type") != "external" {
			if id := attrString(n.Attrs, "id"); id != "" {
				refs.fileIDs[id] = true
			}
			if c := attrString(n.Attrs, "collection"); c == "" || c == ownCollection {
				for _, key := range []string{"__fileName", "alt"} {
					if name := attrString(n.Attrs, key); name != "" {
						refs.filenames[name] = true
					}
				}
			}
		}
		for _, m := range n.Marks {
			if m.Type == "link" {
				refs.addDownloadLink(attrString(m.Attrs, "href"), pageID)
			}
		}
		for _, c := range n.Content {
			walk(c)
		}
	}
	walk(doc)
	return refs, nil
}

func attrString(attrs map[string]any, key string) string {
	s, _ := attrs[key].(string)
	return s
}
