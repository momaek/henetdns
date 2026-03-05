package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/wentx/henetdns/internal/app"
	"github.com/wentx/henetdns/internal/errs"
	"github.com/wentx/henetdns/internal/output"
)

func newLoginCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login and persist session cookie",
		RunE: func(cmd *cobra.Command, args []string) error {
			username := strings.TrimSpace(cfg.Username)
			if username == "" {
				return fmt.Errorf("--username or HE_USERNAME/HE_EMAIL is required: %w", errs.ErrInvalidInput)
			}
			password := cfg.Password
			if password == "" {
				p, err := readPasswordInteractive()
				if err != nil {
					return err
				}
				password = p
			}

			return app.WithRuntime(cfg, func(rt *app.Runtime) error {
				if err := rt.Auth.Login(cmd.Context(), username, password); err != nil {
					return err
				}
				return output.PrintMessage(cmd.OutOrStdout(), "login ok", jsonOut)
			})
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	return cmd
}

func readPasswordInteractive() (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(line), nil
	}
	fmt.Fprint(os.Stderr, "Password: ")
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}
