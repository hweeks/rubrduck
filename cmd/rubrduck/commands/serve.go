package commands

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/hammie/rubrduck/internal/api"
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
		port, _ := cmd.Flags().GetInt("port")
		host, _ := cmd.Flags().GetString("host")

		// Create server configuration
		config := api.ServerConfig{
			Port:               port,
			ReadTimeout:        30 * time.Second,
			WriteTimeout:       30 * time.Second,
			IdleTimeout:        60 * time.Second,
			ShutdownTimeout:    10 * time.Second,
			EnableAuth:         false, // Disable auth for now
			EnableRateLimiting: false, // Disable rate limiting for now
			EnableCORS:         true,
			CORSAllowedOrigins: []string{"*"}, // Allow all origins for development
		}

		// Create server
		server, err := api.NewServer(config)
		if err != nil {
			return fmt.Errorf("failed to create server: %w", err)
		}

		log.Printf("🚀 RubrDuck server starting on %s:%d", host, port)

		// Create context that can be cancelled
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start server in a goroutine
		serverErr := make(chan error, 1)
		go func() {
			if err := server.Start(ctx); err != nil {
				serverErr <- err
			}
		}()

		// Wait for interrupt signal or server error
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt)

		select {
		case err := <-serverErr:
			log.Printf("❌ Server error: %v", err)
			return err
		case <-quit:
			log.Println("🛑 Shutting down server...")
			cancel() // This will gracefully shutdown the server

			// Give the server a moment to shutdown
			time.Sleep(1 * time.Second)
			log.Println("✅ Server stopped")
		}

		return nil
	},
}

func init() {
	serveCmd.Flags().IntP("port", "p", 8080, "Port to run the server on")
	serveCmd.Flags().String("host", "localhost", "Host to bind the server to")
}
