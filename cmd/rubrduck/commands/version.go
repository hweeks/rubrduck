package commands

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// Version is the version of RubrDuck, set at build time using ldflags
	Version = "dev"
	// GitCommit is the git commit hash, set at build time using ldflags
	GitCommit = "unknown"
	// BuildDate is the build date, set at build time using ldflags
	BuildDate = "unknown"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Long:  `Print detailed version information about RubrDuck.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("RubrDuck %s\n", Version)
		fmt.Printf("  Git Commit: %s\n", GitCommit)
		fmt.Printf("  Build Date: %s\n", BuildDate)
		fmt.Printf("  Go Version: %s\n", runtime.Version())
		fmt.Printf("  OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}
