package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/wentx/henetdns/internal/app"
	"github.com/wentx/henetdns/internal/errs"
	"github.com/wentx/henetdns/internal/henet"
	"github.com/wentx/henetdns/internal/output"
)

func newRecordsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "records", Short: "Record operations"}
	cmd.AddCommand(newRecordsListCmd())
	cmd.AddCommand(newRecordsUpsertCmd())
	cmd.AddCommand(newRecordsDeleteCmd())
	return cmd
}

func newRecordsListCmd() *cobra.Command {
	var zone string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List records in a zone",
		RunE: func(cmd *cobra.Command, args []string) error {
			if zone == "" {
				return fmt.Errorf("--zone is required: %w", errs.ErrInvalidInput)
			}
			return app.WithRuntime(cfg, func(rt *app.Runtime) error {
				if err := rt.Auth.EnsureSession(cmd.Context(), cfg.Username); err != nil {
					return err
				}
				zoneID, err := rt.HENet.ResolveZoneID(cmd.Context(), zone)
				if err != nil {
					return err
				}
				records, err := rt.HENet.ListRecords(cmd.Context(), zoneID)
				if err != nil {
					return err
				}
				return output.PrintRecords(cmd.OutOrStdout(), records, jsonOut)
			})
		},
	}
	cmd.Flags().StringVar(&zone, "zone", "", "zone name or zone id")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	return cmd
}

func newRecordsUpsertCmd() *cobra.Command {
	var zone, rrType, name, value string
	var ttl, priority int
	var hasPriority, jsonOut bool

	cmd := &cobra.Command{
		Use:   "upsert",
		Short: "Create record if exact record does not exist",
		RunE: func(cmd *cobra.Command, args []string) error {
			if zone == "" || rrType == "" || name == "" || value == "" {
				return fmt.Errorf("--zone --type --name --value are required: %w", errs.ErrInvalidInput)
			}
			input := henet.RecordInput{Type: rrType, Name: name, Value: value, TTL: ttl, Priority: priority, HasPriority: hasPriority}
			return app.WithRuntime(cfg, func(rt *app.Runtime) error {
				if err := rt.Auth.EnsureSession(cmd.Context(), cfg.Username); err != nil {
					return err
				}
				zoneID, err := rt.HENet.ResolveZoneID(cmd.Context(), zone)
				if err != nil {
					return err
				}
				if err := rt.HENet.UpsertRecord(cmd.Context(), zoneID, input); err != nil {
					return err
				}
				return output.PrintMessage(cmd.OutOrStdout(), "upsert ok", jsonOut)
			})
		},
	}
	cmd.Flags().StringVar(&zone, "zone", "", "zone name or zone id")
	cmd.Flags().StringVar(&rrType, "type", "", "record type (A/AAAA/TXT/CNAME/MX)")
	cmd.Flags().StringVar(&name, "name", "", "record name")
	cmd.Flags().StringVar(&value, "value", "", "record value")
	cmd.Flags().IntVar(&ttl, "ttl", 300, "record ttl")
	cmd.Flags().IntVar(&priority, "priority", 10, "priority for MX")
	cmd.Flags().BoolVar(&hasPriority, "priority-set", false, "set when --priority should be used for MX")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	return cmd
}

func newRecordsDeleteCmd() *cobra.Command {
	var zone, rrType, name, value string
	var priority int
	var hasPriority, jsonOut bool

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete exact matching record",
		RunE: func(cmd *cobra.Command, args []string) error {
			if zone == "" || rrType == "" || name == "" || value == "" {
				return fmt.Errorf("--zone --type --name --value are required: %w", errs.ErrInvalidInput)
			}
			input := henet.RecordInput{Type: rrType, Name: name, Value: value, Priority: priority, HasPriority: hasPriority}
			return app.WithRuntime(cfg, func(rt *app.Runtime) error {
				if err := rt.Auth.EnsureSession(cmd.Context(), cfg.Username); err != nil {
					return err
				}
				zoneID, err := rt.HENet.ResolveZoneID(cmd.Context(), zone)
				if err != nil {
					return err
				}
				if err := rt.HENet.DeleteRecord(cmd.Context(), zoneID, input); err != nil {
					return err
				}
				return output.PrintMessage(cmd.OutOrStdout(), "delete ok", jsonOut)
			})
		},
	}
	cmd.Flags().StringVar(&zone, "zone", "", "zone name or zone id")
	cmd.Flags().StringVar(&rrType, "type", "", "record type (A/AAAA/TXT/CNAME/MX)")
	cmd.Flags().StringVar(&name, "name", "", "record name")
	cmd.Flags().StringVar(&value, "value", "", "record value")
	cmd.Flags().IntVar(&priority, "priority", 10, "priority for MX")
	cmd.Flags().BoolVar(&hasPriority, "priority-set", false, "set when --priority should be used for MX")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	return cmd
}
