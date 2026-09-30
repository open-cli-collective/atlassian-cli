package api

// PreserveDescriptionMedia returns doc with the media blocks (attached images
// and files) from existing appended after its content. Markdown and wiki
// input have no syntax for Jira media, so a replacement built from text would
// otherwise drop every inline attachment reference from the description.
// Media nested in containers such as tables or panels is kept at the top level,
// because the container it sat in is being replaced. doc is modified in place,
// and the appended nodes are shared with existing rather than copied.
func PreserveDescriptionMedia(doc *ADFDocument, existing *Description) *ADFDocument {
	if existing == nil || existing.ADF == nil {
		return doc
	}
	media := collectMediaBlocks(existing.ADF.Content, nil)
	if len(media) == 0 {
		return doc
	}
	if doc == nil {
		doc = &ADFDocument{Type: "doc", Version: 1}
	}
	doc.Content = append(doc.Content, media...)
	return doc
}

func collectMediaBlocks(nodes []*ADFNode, found []*ADFNode) []*ADFNode {
	for _, n := range nodes {
		if n == nil {
			continue
		}
		switch n.Type {
		case "mediaSingle", "mediaGroup":
			found = append(found, n)
		default:
			found = collectMediaBlocks(n.Content, found)
		}
	}
	return found
}
