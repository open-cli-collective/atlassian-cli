// Package attachments provides CLI commands for managing Jira issue attachments.
package attachments

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
)

// Register registers the attachments commands
func Register(parent *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:     "attachments",
		Aliases: []string{"attachment", "att"},
		Short:   "Manage issue attachments",
		Long:    "Commands for listing, adding, downloading, and deleting issue attachments.",
	}

	cmd.AddCommand(newListCmd(opts))
	cmd.AddCommand(newAddCmd(opts))
	cmd.AddCommand(newGetCmd(opts))
	cmd.AddCommand(newDeleteCmd(opts))

	parent.AddCommand(cmd)
}

func newListCmd(opts *root.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list <issue-key>",
		Aliases: []string{"ls"},
		Short:   "List attachments on an issue",
		Long:    "List all attachments on a Jira issue.",
		Example: `  # List attachments
  jtk attachments list PROJ-123

  # Output as JSON
  jtk attachments list PROJ-123 -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), opts, args[0])
		},
	}

	return cmd
}

func runList(ctx context.Context, opts *root.Options, issueKey string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	attachments, err := client.GetIssueAttachments(ctx, issueKey)
	if err != nil {
		return err
	}

	if len(attachments) == 0 {
		model := jtkpresent.AttachmentPresenter{}.PresentEmpty(issueKey)
		out := present.Render(model, opts.RenderStyle())
		_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
		return nil
	}

	if v.Format == view.FormatJSON {
		data := make([]map[string]any, 0, len(attachments))
		for _, att := range attachments {
			data = append(data, map[string]any{
				"id":       att.ID.String(),
				"filename": att.Filename,
				"size":     att.Size,
				"mimeType": att.MimeType,
				"created":  att.Created,
				"author":   att.Author.DisplayName,
				"content":  att.Content,
			})
		}

		return v.JSON(data)
	}

	// Text path: presenter → render → write
	model := jtkpresent.AttachmentPresenter{}.PresentList(attachments)
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
}

func newAddCmd(opts *root.Options) *cobra.Command {
	var files []string

	cmd := &cobra.Command{
		Use:   "add <issue-key>",
		Short: "Add attachments to an issue",
		Long:  "Upload one or more files as attachments to a Jira issue.",
		Example: `  # Add a single file
  jtk attachments add PROJ-123 --file screenshot.png

  # Add multiple files
  jtk attachments add PROJ-123 --file doc.pdf --file image.png`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdd(cmd.Context(), opts, args[0], files)
		},
	}

	cmd.Flags().StringArrayVarP(&files, "file", "f", nil, "File(s) to attach (can be specified multiple times)")
	_ = cmd.MarkFlagRequired("file")

	return cmd
}

func runAdd(ctx context.Context, opts *root.Options, issueKey string, files []string) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	for _, filePath := range files {
		// Expand path
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			return fmt.Errorf("invalid file path %s: %w", filePath, err)
		}

		// Check file exists
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", filePath)
		}

		attachments, err := client.AddAttachment(ctx, issueKey, absPath)
		if err != nil {
			return fmt.Errorf("uploading %s: %w", filepath.Base(filePath), err)
		}

		for _, att := range attachments {
			model := jtkpresent.AttachmentPresenter{}.PresentUploaded(att.Filename, att.ID.String(), api.FormatFileSize(att.Size))
			out := present.Render(model, opts.RenderStyle())
			_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
		}
	}

	return nil
}

func newGetCmd(opts *root.Options) *cobra.Command {
	var outputPath string

	cmd := &cobra.Command{
		Use:     "get <attachment-id>",
		Aliases: []string{"download"},
		Short:   "Download an attachment",
		Long:    "Download an attachment by its ID.",
		Example: `  # Download to current directory
  jtk attachments get 12345

  # Download to specific directory
  jtk attachments get 12345 --output ./downloads/

  # Download with custom filename
  jtk attachments get 12345 --output ./downloads/renamed.pdf`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGet(cmd.Context(), opts, args[0], outputPath)
		},
	}

	cmd.Flags().StringVarP(&outputPath, "output", "o", ".", "Output path (directory or filename)")

	return cmd
}

func runGet(ctx context.Context, opts *root.Options, attachmentID, outputPath string) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	// Get attachment metadata
	attachment, err := client.GetAttachment(ctx, attachmentID)
	if err != nil {
		return fmt.Errorf("getting attachment: %w", err)
	}

	// Download the file
	if err := client.DownloadAttachment(ctx, attachment, outputPath); err != nil {
		return fmt.Errorf("downloading attachment: %w", err)
	}

	// Determine actual output path for message
	actualPath := outputPath
	if info, err := os.Stat(outputPath); err == nil && info.IsDir() {
		actualPath = filepath.Join(outputPath, attachment.Filename)
	}

	model := jtkpresent.AttachmentPresenter{}.PresentDownloaded(actualPath, api.FormatFileSize(attachment.Size))
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	return nil
}

func newDeleteCmd(opts *root.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <attachment-id>",
		Aliases: []string{"rm"},
		Short:   "Delete an attachment",
		Long:    "Delete an attachment by its ID.",
		Example: `  # Delete an attachment
  jtk attachments delete 12345`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(cmd.Context(), opts, args[0])
		},
	}

	return cmd
}

func runDelete(ctx context.Context, opts *root.Options, attachmentID string) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	if err := client.DeleteAttachment(ctx, attachmentID); err != nil {
		return fmt.Errorf("deleting attachment: %w", err)
	}

	model := jtkpresent.AttachmentPresenter{}.PresentDeleted(attachmentID)
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	return nil
}
