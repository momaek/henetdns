package cli

import (
	"github.com/spf13/cobra"
	"github.com/momaek/henetdns/internal/config"
)

var cfg config.Config

func Execute(version string) error {
	root := newRootCmd(version)
	return root.Execute()
}

func newRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "henetdns",
		Short:   "CLI for Hurricane Electric hosted DNS",
		Version: version,
		// Runtime failures (auth, no matching record, ...) are not usage
		// mistakes; main prints the error once with an exit code.
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			config.ApplyEnv(&cfg)
			return config.ValidateCommon(cfg)
		},
	}

	cmd.PersistentFlags().StringVar(&cfg.BaseURL, "base-url", "", "he.net DNS base URL (env: HENETDNS_BASE_URL)")
	cmd.PersistentFlags().StringVar(&cfg.DataDir, "data-dir", "", "data directory for session and cache (env: HENETDNS_DATA_DIR)")
	cmd.PersistentFlags().StringVar(&cfg.Username, "username", "", "account username (env: HE_USERNAME, fallback: HE_EMAIL)")
	cmd.PersistentFlags().StringVar(&cfg.Email, "email", "", "DEPRECATED alias of --username (env: HE_EMAIL)")
	_ = cmd.PersistentFlags().MarkDeprecated("email", "use --username (or HE_USERNAME) instead")
	cmd.PersistentFlags().StringVar(&cfg.Password, "password", "", "account password (env: HE_PASS)")
	cmd.PersistentFlags().DurationVar(&cfg.Timeout, "timeout", 0, "HTTP timeout, e.g. 20s (env: HENETDNS_TIMEOUT)")

	cmd.AddCommand(newLoginCmd())
	cmd.AddCommand(newZonesCmd())
	cmd.AddCommand(newRecordsCmd())
	cmd.AddCommand(newVersionCmd(version))
	cmd.AddCommand(newUpgradeCmd(version))
	return cmd
}
