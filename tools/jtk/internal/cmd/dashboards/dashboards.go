// Package dashboards provides CLI commands for managing Jira dashboards.
package dashboards

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

// Register registers the dashboards commands
func Register(parent *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:     "dashboards",
		Aliases: []string{"dashboard", "dash"},
		Short:   "Manage dashboards",
		Long:    "Commands for listing, creating, and managing Jira dashboards and their gadgets.",
	}

	cmd.AddCommand(newListCmd(opts))
	cmd.AddCommand(newGetCmd(opts))
	cmd.AddCommand(newCreateCmd(opts))
	cmd.AddCommand(newDeleteCmd(opts))
	cmd.AddCommand(newGadgetsCmd(opts))

	parent.AddCommand(cmd)
}

func newListCmd(opts *root.Options) *cobra.Command {
	var search string
	var maxResults int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List dashboards",
		Long:  "List accessible dashboards. Use --search to filter by name.",
		Example: `  jtk dashboards list
  jtk dashboards list --search "Sprint"
  jtk dashboards list --max 10`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runList(opts, search, maxResults)
		},
	}

	cmd.Flags().StringVar(&search, "search", "", "Search dashboards by name")
	cmd.Flags().IntVar(&maxResults, "max", 50, "Maximum number of results")

	return cmd
}

func runList(opts *root.Options, search string, maxResults int) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	if search != "" {
		result, err := client.SearchDashboards(search, maxResults)
		if err != nil {
			return err
		}

		if len(result.Values) == 0 {
			v.Info("No dashboards found matching %q", search)
			return nil
		}

		if opts.Output == "json" {
			return v.JSON(result.Values)
		}

		return renderDashboardTable(v, result.Values)
	}

	result, err := client.GetDashboards(0, maxResults)
	if err != nil {
		return err
	}

	if len(result.Dashboards) == 0 {
		v.Info("No dashboards found")
		return nil
	}

	if opts.Output == "json" {
		return v.JSON(result.Dashboards)
	}

	return renderDashboardTable(v, result.Dashboards)
}

func newGetCmd(opts *root.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <dashboard-id>",
		Short: "Get dashboard details",
		Long:  "Get details of a specific dashboard including its gadgets.",
		Example: `  jtk dashboards get 10001
  jtk dashboards get 10001 -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runGet(opts, args[0])
		},
	}

	return cmd
}

func runGet(opts *root.Options, dashboardID string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	dash, err := client.GetDashboard(dashboardID)
	if err != nil {
		return err
	}

	// Also get gadgets
	gadgets, err := client.GetDashboardGadgets(dashboardID)
	if err != nil {
		return fmt.Errorf("failed to get gadgets: %w", err)
	}

	if opts.Output == "json" {
		return v.JSON(map[string]interface{}{
			"dashboard": dash,
			"gadgets":   gadgets.Gadgets,
		})
	}

	v.Println("ID:          %s", dash.ID)
	v.Println("Name:        %s", dash.Name)
	if dash.Description != "" {
		v.Println("Description: %s", dash.Description)
	}
	if dash.Owner != nil {
		v.Println("Owner:       %s", dash.Owner.DisplayName)
	}
	if dash.View != "" {
		v.Println("URL:         %s", dash.View)
	}

	if len(gadgets.Gadgets) > 0 {
		v.Println("")
		v.Println("Gadgets (%d):", len(gadgets.Gadgets))

		headers := []string{"ID", "TITLE", "MODULE", "POSITION"}
		var rows [][]string

		for _, g := range gadgets.Gadgets {
			pos := fmt.Sprintf("row=%d col=%d", g.Position.Row, g.Position.Column)
			rows = append(rows, []string{
				strconv.Itoa(g.ID),
				g.Title,
				g.ModuleID,
				pos,
			})
		}
		return v.Table(headers, rows)
	}

	v.Info("\nNo gadgets on this dashboard")
	return nil
}

func newCreateCmd(opts *root.Options) *cobra.Command {
	var name string
	var description string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new dashboard",
		Long:  "Create a new Jira dashboard.",
		Example: `  jtk dashboards create --name "My Dashboard"
  jtk dashboards create --name "Sprint Board" --description "Sprint tracking"`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runCreate(opts, name, description)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Dashboard name (required)")
	cmd.Flags().StringVar(&description, "description", "", "Dashboard description")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func runCreate(opts *root.Options, name, description string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	req := api.CreateDashboardRequest{
		Name:             name,
		Description:      description,
		EditPermissions:  []api.SharePerm{},
		SharePermissions: []api.SharePerm{},
	}

	dash, err := client.CreateDashboard(req)
	if err != nil {
		return err
	}

	if opts.Output == "json" {
		return v.JSON(dash)
	}

	v.Success("Created dashboard %s (%s)", dash.Name, dash.ID)
	if dash.View != "" {
		v.Info("URL: %s", dash.View)
	}

	return nil
}

func newDeleteCmd(opts *root.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <dashboard-id>",
		Short:   "Delete a dashboard",
		Long:    "Delete a Jira dashboard by its ID.",
		Example: `  jtk dashboards delete 10001`,
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runDelete(opts, args[0])
		},
	}

	return cmd
}

func runDelete(opts *root.Options, dashboardID string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	if err := client.DeleteDashboard(dashboardID); err != nil {
		return err
	}

	if opts.Output == "json" {
		return v.JSON(map[string]string{"status": "deleted", "dashboardId": dashboardID})
	}

	v.Success("Deleted dashboard %s", dashboardID)
	return nil
}

func newGadgetsCmd(opts *root.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gadgets",
		Short: "Manage dashboard gadgets",
		Long:  "Commands for listing and removing gadgets on dashboards.",
	}

	cmd.AddCommand(newGadgetsListCmd(opts))
	cmd.AddCommand(newGadgetsRemoveCmd(opts))

	return cmd
}

func newGadgetsListCmd(opts *root.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <dashboard-id>",
		Short: "List gadgets on a dashboard",
		Long:  "List all gadgets on a specific dashboard.",
		Example: `  jtk dashboards gadgets list 10001
  jtk dashboards gadgets list 10001 -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runGadgetsList(opts, args[0])
		},
	}

	return cmd
}

func runGadgetsList(opts *root.Options, dashboardID string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	result, err := client.GetDashboardGadgets(dashboardID)
	if err != nil {
		return err
	}

	if len(result.Gadgets) == 0 {
		v.Info("No gadgets on dashboard %s", dashboardID)
		return nil
	}

	if opts.Output == "json" {
		return v.JSON(result.Gadgets)
	}

	headers := []string{"ID", "TITLE", "MODULE", "POSITION"}
	var rows [][]string

	for _, g := range result.Gadgets {
		pos := fmt.Sprintf("row=%d col=%d", g.Position.Row, g.Position.Column)
		rows = append(rows, []string{
			strconv.Itoa(g.ID),
			g.Title,
			g.ModuleID,
			pos,
		})
	}

	return v.Table(headers, rows)
}

func newGadgetsRemoveCmd(opts *root.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove <dashboard-id> <gadget-id>",
		Short:   "Remove a gadget from a dashboard",
		Long:    "Remove a gadget from a dashboard by its ID.",
		Example: `  jtk dashboards gadgets remove 10001 42`,
		Args:    cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			gadgetID, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid gadget ID: %s", args[1])
			}
			return runGadgetsRemove(opts, args[0], gadgetID)
		},
	}

	return cmd
}

func runGadgetsRemove(opts *root.Options, dashboardID string, gadgetID int) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	if err := client.RemoveDashboardGadget(dashboardID, gadgetID); err != nil {
		return err
	}

	if opts.Output == "json" {
		return v.JSON(map[string]interface{}{
			"status":      "removed",
			"dashboardId": dashboardID,
			"gadgetId":    gadgetID,
		})
	}

	v.Success("Removed gadget %d from dashboard %s", gadgetID, dashboardID)
	return nil
}

type viewWriter interface {
	Table(headers []string, rows [][]string) error
}

func renderDashboardTable(v viewWriter, dashboards []api.Dashboard) error {
	headers := []string{"ID", "NAME", "OWNER", "FAVOURITE"}
	var rows [][]string

	for _, d := range dashboards {
		owner := ""
		if d.Owner != nil {
			owner = d.Owner.DisplayName
		}
		fav := ""
		if d.IsFavourite {
			fav = "yes"
		}
		rows = append(rows, []string{d.ID, d.Name, owner, fav})
	}

	return v.Table(headers, rows)
}
