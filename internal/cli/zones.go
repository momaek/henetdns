package cli

import (
	"github.com/spf13/cobra"

	"github.com/wentx/henetdns/internal/app"
	"github.com/wentx/henetdns/internal/output"
)

func newZonesCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "zones", Short: "Zone operations"}
	cmd.AddCommand(newZonesListCmd())
	return cmd
}

func newZonesListCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List zones",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.WithRuntime(cfg, func(rt *app.Runtime) error {
				if err := rt.Auth.EnsureSession(cmd.Context(), cfg.Username); err != nil {
					return err
				}
				zones, err := rt.HENet.ListZones(cmd.Context())
				if err != nil {
					return err
				}
				return output.PrintZones(cmd.OutOrStdout(), zones, jsonOut)
			})
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	return cmd
}
