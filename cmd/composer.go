package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/thesmartyking/herd/pkg/php"
)

var composerCmd = &cobra.Command{
	Use:   "composer [args...]",
	Short: "Run Composer with the current Herd site's PHP version",
	Long: `Runs Composer using the same PHP CLI version pinned for the Herd site
that owns your current directory (not the system default php).

Examples:
  cd ~/Code/myapp
  herd composer install
  herd composer update
  herd composer --site=myapp require laravel/pint`,
	DisableFlagParsing: true,
	Args:               cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		args, siteName := php.ParseSiteFlag(args)
		if len(args) == 1 && (args[0] == "help" || args[0] == "--help" || args[0] == "-h") {
			return cmd.Help()
		}
		if len(args) == 0 {
			fmt.Println("Usage: herd composer <composer-args...>")
			fmt.Println("Example: herd composer install")
			site, bin, err := php.ResolveSitePHP(siteName)
			if err == nil {
				fmt.Printf("\nCurrent site: %s → PHP %s (%s)\n", site.Name, site.PHPVersion, bin)
			}
			return nil
		}

		err := php.ExecSiteComposer(siteName, args)
		if code := php.ExitCode(err); code >= 0 {
			os.Exit(code)
		}
		return err
	},
}

func init() {
	rootCmd.AddCommand(composerCmd)
}
