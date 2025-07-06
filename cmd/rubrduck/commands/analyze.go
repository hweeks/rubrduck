package commands

import (
	"fmt"

	"github.com/hammie/rubrduck/internal/project"
	"github.com/spf13/cobra"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze [path]",
	Short: "Analyze project structure and configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}
		analysis, err := project.Analyze(path)
		if err != nil {
			return err
		}

		fmt.Printf("Project root: %s\n", analysis.Root)
		if len(analysis.Languages) > 0 {
			fmt.Println("Languages:")
			for lang, count := range analysis.Languages {
				fmt.Printf("  %s (%d files)\n", lang, count)
			}
		}
		if len(analysis.Frameworks) > 0 {
			fmt.Println("Frameworks:")
			for _, fw := range analysis.Frameworks {
				fmt.Printf("  %s\n", fw)
			}
		}
		if len(analysis.ConfigFiles) > 0 {
			fmt.Println("Config files:")
			for _, cfg := range analysis.ConfigFiles {
				fmt.Printf("  %s\n", cfg)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
}
