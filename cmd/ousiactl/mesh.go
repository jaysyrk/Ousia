package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var meshCmd = &cobra.Command{
	Use:   "mesh",
	Short: "Manage service mesh instances",
}

var meshListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered service instances in the mesh",
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := client.Get(adminURL + "/api/mesh/services")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err == nil {
			pretty, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(pretty))
		} else {
			fmt.Println(string(body))
		}
	},
}

func init() {
	rootCmd.AddCommand(meshCmd)
	meshCmd.AddCommand(meshListCmd)
}
