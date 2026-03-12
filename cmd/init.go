package cmd

import (
	"fmt"
	"os"

	"github.com/allskar/llmux/internal/config"
	"github.com/allskar/llmux/internal/shell"
	"github.com/spf13/cobra"
)

var printFlag bool

var initCmd = &cobra.Command{
	Use:       "init [shell]",
	Short:     "Install shell integration",
	Long:      "Appends the claude() wrapper to your shell rc file.\nUse --print to output the function without modifying any files.",
	Args:      cobra.ExactArgs(1),
	ValidArgs:    []string{"zsh", "bash", "fish"},
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		bin, err := os.Executable()
		if err != nil {
			return fmt.Errorf("could not determine executable path: %w", err)
		}

		cfg, err := config.Load()
		if err != nil {
			cfg = &config.Config{}
		}

		if printFlag {
			out, err := shell.Generate(bin, args[0], cfg.ShortAlias)
			if err != nil {
				return err
			}
			fmt.Println(out)
			return nil
		}

		rc, err := shell.Install(bin, args[0])
		if err != nil {
			return err
		}
		fmt.Printf("Added llmux shell integration to %s\n", rc)
		fmt.Println("Restart your shell or run: source", rc)
		return nil
	},
}

func init() {
	initCmd.Flags().BoolVar(&printFlag, "print", false, "Print the shell function to stdout instead of installing")
	rootCmd.AddCommand(initCmd)
}
