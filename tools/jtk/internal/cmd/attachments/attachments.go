// Package attachments provides CLI commands for managing Jira issue attachments.
package attachments

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

func noFieldFetch(_ context.Context) ([]api.Field, error) { return nil, nil }

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
	var fieldsFlag string

	cmd := &cobra.Command{
		Use:     "list <issue-key>",
		Aliases: []string{"ls"},
		Short:   "List attachments on an issue",
		Long:    "List all attachments on a Jira issue.",
		Example: `  # List attachments
  jtk attachments list PROJ-123
  jtk attachments list PROJ-123 --extended
  jtk attachments list PROJ-123 --id`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), opts, args[0], fieldsFlag)
		},
	}

	cmd.Flags().StringVar(&fieldsFlag, "fields", "", "Comma-separated display columns")

	return cmd
}

func runList(ctx context.Context, opts *root.Options, issueKey, fieldsFlag string) error {
	v := opts.View()
	idOnly := opts.EmitIDOnly()

	if !idOnly && fieldsFlag != "" && v.Format == view.FormatJSON {
		return jtkpresent.ErrFieldsWithJSON
	}

	var selected []projection.ColumnSpec
	var projected bool
	if !idOnly {
		var err error
		selected, projected, err = projection.Resolve(
			ctx,
			jtkpresent.AttachmentListSpec,
			opts.IsExtended(),
			fieldsFlag,
			noFieldFetch,
			"attachments list",
		)
		if err != nil {
			return err
		}
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	attachments, err := client.GetIssueAttachments(ctx, issueKey)
	if err != nil {
		return err
	}

	if idOnly {
		ids := make([]string, len(attachments))
		for i, a := range attachments {
			ids[i] = a.ID.String()
		}
		return jtkpresent.EmitIDs(opts, ids)
	}

	if len(attachments) == 0 {
		return jtkpresent.Emit(opts, jtkpresent.AttachmentPresenter{}.PresentEmpty(issueKey))
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

	model := jtkpresent.AttachmentPresenter{}.PresentList(attachments, opts.IsExtended())
	if projected {
		projection.ApplyToTableInModel(model, selected)
	}
	return jtkpresent.Emit(opts, model)
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

	var allAttachments []api.Attachment
	for _, filePath := range files {
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			return fmt.Errorf("invalid file path %s: %w", filePath, err)
		}

		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", filePath)
		}

		attachments, err := client.AddAttachment(ctx, issueKey, absPath)
		if err != nil {
			// Emit results for files that succeeded before returning the error
			if len(allAttachments) > 0 {
				if opts.EmitIDOnly() {
					ids := make([]string, len(allAttachments))
					for i, a := range allAttachments {
						ids[i] = a.ID.String()
					}
					_ = jtkpresent.EmitIDs(opts, ids)
				} else {
					_ = jtkpresent.Emit(opts, jtkpresent.AttachmentPresenter{}.PresentList(allAttachments, opts.IsExtended()))
				}
			}
			return fmt.Errorf("uploading %s: %w", filepath.Base(filePath), err)
		}

		allAttachments = append(allAttachments, attachments...)
	}

	if opts.EmitIDOnly() {
		ids := make([]string, len(allAttachments))
		for i, a := range allAttachments {
			ids[i] = a.ID.String()
		}
		return jtkpresent.EmitIDs(opts, ids)
	}

	return jtkpresent.Emit(opts, jtkpresent.AttachmentPresenter{}.PresentList(allAttachments, opts.IsExtended()))
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

	return jtkpresent.Emit(opts, jtkpresent.AttachmentPresenter{}.PresentDownloaded(attachment.ID.String(), actualPath, attachment.Size))
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

	return jtkpresent.Emit(opts, jtkpresent.AttachmentPresenter{}.PresentDeleted(attachmentID))
}
