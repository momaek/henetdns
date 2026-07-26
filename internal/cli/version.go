package cli

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/momaek/henetdns/internal/update"
)

func newVersionCmd(version string) *cobra.Command {
	var check, jsonOut bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version and optionally check for a newer release",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !check {
				if jsonOut {
					return printJSON(cmd, map[string]any{"version": version})
				}
				fmt.Fprintf(cmd.OutOrStdout(), "henetdns version %s\n", version)
				return nil
			}

			client := &http.Client{Timeout: cfg.Timeout}
			latest, err := update.LatestTag(cmd.Context(), client)
			if err != nil {
				return err
			}
			isRelease := update.IsRelease(version)
			updateAvailable := isRelease && update.Compare(version, latest) < 0

			if jsonOut {
				out := map[string]any{
					"version": version,
					"latest":  latest,
				}
				if isRelease {
					out["update_available"] = updateAvailable
				}
				return printJSON(cmd, out)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "henetdns version %s\n", version)
			fmt.Fprintf(cmd.OutOrStdout(), "latest release: %s\n", latest)
			switch {
			case !isRelease:
				fmt.Fprintln(cmd.OutOrStdout(), "current version is not a release build; cannot compare")
			case updateAvailable:
				fmt.Fprintln(cmd.OutOrStdout(), "update available — run `henetdns upgrade`")
			default:
				fmt.Fprintln(cmd.OutOrStdout(), "up to date")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "check GitHub for the latest release")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	return cmd
}

func printJSON(cmd *cobra.Command, v any) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
