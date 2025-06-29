package commands

import (
	"github.com/spf13/cobra"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the API server for IDE integrations",
	Long: `Start the RubrDuck API server that IDE extensions can connect to.
	
The server provides a REST API and WebSocket connections for real-time
communication with IDE extensions.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Implement API server
		return nil
	},
}

func init() {
	serveCmd.Flags().IntP("port", "p", 8080, "Port to run the server on")
	serveCmd.Flags().String("host", "localhost", "Host to bind the server to")
}
