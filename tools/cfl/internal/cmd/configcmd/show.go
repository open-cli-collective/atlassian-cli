package configcmd

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/credstore"
	"github.com/open-cli-collective/atlassian-go/keyring"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
	"github.com/open-cli-collective/confluence-cli/internal/config"
	cflpresent "github.com/open-cli-collective/confluence-cli/internal/present"
)

func newShowCmd(opts *root.Options) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Long: `Display the current cfl configuration.

The API token value is never displayed — only whether one is configured,
where it resolves from, and the OS keyring backend in use. Token/keyring
reporting is authoritative.

Note: the non-secret rows (URL, email, etc.) reflect environment
variables and the legacy per-tool file ONLY. A value set solely in the
shared store at ~/.config/atlassian-cli/config.yml is shown as "not set"
here even though the tool resolves and uses it at runtime — run a real
command to confirm effective configuration.`,
		Example: `  # Show current configuration
  cfl config show`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runShow(opts)
		},
	}
}

func runShow(opts *root.Options) error {
	configPath := config.DefaultConfigPath()

	// Load config file (if exists)
	fileCfg, fileErr := config.Load(configPath)
	if fileErr != nil {
		fileCfg = &config.Config{}
	}

	// Non-secret keyring description (non-migrating: show stays usable
	// during an unresolved §1.8 conflict). The token VALUE is never
	// shown — presence + source only (§1.12).
	kr, krErr := keyring.InspectForTool(credstore.ToolCFL)

	proj := config.ProjectShow(configPath, fileCfg, fileErr, kr, krErr)
	return cflpresent.Emit(opts, cflpresent.ConfigShowPresenter{}.PresentDetail(proj))
}
