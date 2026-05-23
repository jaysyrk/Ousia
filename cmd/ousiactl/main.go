package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var adminURL string
var client = &http.Client{Timeout: 5 * time.Second}

var rootCmd = &cobra.Command{
	Use:	"ousiactl",
	Short:	"ousiactl is the CLI for the Ousia Service Mesh",
	Long:	`A fast and flexible CLI for managing the Ousia Edge Gateway and Service Mesh.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&adminURL, "admin-url", "http://127.0.0.1:9000", "Control Plane Admin API URL")
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(metricsCmd)
}

var statusCmd = &cobra.Command{
	Use:	"status",
	Short:	"Check the health of the Ousia Gateway",
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := client.Get(adminURL + "/api/health")
		if err != nil {
			fmt.Printf("Error connecting to gateway: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("Gateway returned status %d\n", resp.StatusCode)
			os.Exit(1)
		}

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			fmt.Printf("Raw response: %s\n", string(body))
			return
		}

		fmt.Println("Ousia Gateway Status:")
		for k, v := range result {
			fmt.Printf("  %s: %v\n", k, v)
		}
	},
}

var metricsCmd = &cobra.Command{
	Use:	"metrics",
	Short:	"View consolidated mesh and gateway metrics",
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := client.Get(adminURL + "/api/stats")
		if err != nil {
			fmt.Printf("Error fetching metrics: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("Gateway returned status %d\n", resp.StatusCode)
			os.Exit(1)
		}

		body, _ := io.ReadAll(resp.Body)

		var parsed map[string]interface{}
		if err := json.Unmarshal(body, &parsed); err == nil {
			pretty, _ := json.MarshalIndent(parsed, "", "  ")
			fmt.Println(string(pretty))
		} else {
			fmt.Println(string(body))
		}
	},
}
