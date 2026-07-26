package cli

import (
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"

	"github.com/momaek/henetdns/internal/errs"
	"github.com/momaek/henetdns/internal/output"
	"github.com/momaek/henetdns/internal/update"
)

func newUpgradeCmd(version string) *cobra.Command {
	var force, jsonOut bool
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Replace this binary with the latest GitHub release",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Downloads can outlast the regular HTTP timeout.
			client := &http.Client{Timeout: 5 * time.Minute}

			latest, err := update.LatestTag(cmd.Context(), client)
			if err != nil {
				return err
			}
			if !update.IsRelease(version) {
				if !force {
					return fmt.Errorf("current version %q is not a release build; re-run with --force to replace this binary with %s: %w", version, latest, errs.ErrInvalidInput)
				}
			} else if update.Compare(version, latest) >= 0 && !force {
				return output.PrintMessage(cmd.OutOrStdout(), fmt.Sprintf("already up to date (%s)", version), jsonOut)
			}

			path, err := update.Upgrade(cmd.Context(), client, latest)
			if err != nil {
				return err
			}
			return output.PrintMessage(cmd.OutOrStdout(), fmt.Sprintf("upgraded to %s (%s)", latest, path), jsonOut)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "reinstall even if already up to date or not a release build")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	return cmd
}
