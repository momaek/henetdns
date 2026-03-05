package cli

import (
	"github.com/spf13/cobra"
	"github.com/wentx/henetdns/internal/config"
)

var cfg config.Config

func Execute() error {
	root := newRootCmd()
	return root.Execute()
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "henetdns",
		Short: "CLI for Hurricane Electric hosted DNS",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			config.ApplyEnv(&cfg)
			return config.ValidateCommon(cfg)
		},
	}

	cmd.PersistentFlags().StringVar(&cfg.BaseURL, "base-url", "", "he.net DNS base URL (env: HENETDNS_BASE_URL)")
	cmd.PersistentFlags().StringVar(&cfg.DBPath, "db-path", "", "sqlite db path (env: HENETDNS_DB_PATH)")
	cmd.PersistentFlags().StringVar(&cfg.Username, "username", "", "account username (env: HE_USERNAME, fallback: HE_EMAIL)")
	cmd.PersistentFlags().StringVar(&cfg.Email, "email", "", "DEPRECATED alias of --username (env: HE_EMAIL)")
	_ = cmd.PersistentFlags().MarkDeprecated("email", "use --username (or HE_USERNAME) instead")
	cmd.PersistentFlags().StringVar(&cfg.Password, "password", "", "account password (env: HE_PASS)")
	cmd.PersistentFlags().DurationVar(&cfg.Timeout, "timeout", 0, "HTTP timeout, e.g. 20s (env: HENETDNS_TIMEOUT)")

	cmd.AddCommand(newLoginCmd())
	cmd.AddCommand(newZonesCmd())
	cmd.AddCommand(newRecordsCmd())
	return cmd
}
