package cli

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ashvinbhat/yoke/internal/config"
	"github.com/spf13/cobra"
)

//go:embed agents_doc.md
var agentsDoc string

var docsStdout bool

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Write the usage doc to ~/.yoke and print its path",
	Long: `Writes yoke's canonical usage documentation to ~/.yoke/AGENTS.md
and prints the path.

The doc is embedded in the binary, so this always reflects the installed
version's actual capabilities. Tools that integrate with yoke (agent
workspaces, editors) should symlink or read this file rather than
maintaining their own descriptions of yoke's interface.

Examples:
  yoke docs            # sync ~/.yoke/AGENTS.md, print path
  yoke docs --stdout   # print the doc itself`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if docsStdout {
			fmt.Print(agentsDoc)
			return nil
		}
		path, err := writeAgentsDoc()
		if err != nil {
			return err
		}
		fmt.Println(path)
		return nil
	},
}

// writeAgentsDoc writes the embedded doc to ~/.yoke/AGENTS.md, returning
// the path. Idempotent; always overwrites so the file tracks the installed
// binary's version.
func writeAgentsDoc() (string, error) {
	if err := config.EnsureDir(); err != nil {
		return "", fmt.Errorf("ensure yoke dir: %w", err)
	}
	path := filepath.Join(config.YokeDir(), "AGENTS.md")
	if err := os.WriteFile(path, []byte(agentsDoc), 0o644); err != nil {
		return "", fmt.Errorf("write agents doc: %w", err)
	}
	return path, nil
}

func init() {
	docsCmd.Flags().BoolVar(&docsStdout, "stdout", false, "Print the doc to stdout instead of writing the file")
	rootCmd.AddCommand(docsCmd)
}
