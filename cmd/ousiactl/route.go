package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

var routeCmd = &cobra.Command{
	Use:	"route",
	Short:	"Manage routing rules",
}

var routeListCmd = &cobra.Command{
	Use:	"list",
	Short:	"List all routes",
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := client.Get(adminURL + "/api/routes")
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

var routeAddCmd = &cobra.Command{
	Use:	"add",
	Short:	"Add a new route",
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetString("id")
		vhost, _ := cmd.Flags().GetString("virtual-host")
		upstream, _ := cmd.Flags().GetString("upstream")
		prefix, _ := cmd.Flags().GetString("path-prefix")

		payload := map[string]interface{}{
			"id":		id,
			"virtual_host":	vhost,
			"upstream":	upstream,
			"priority":	100,
		}
		if prefix != "" {
			payload["path_prefix"] = prefix
		}

		data, _ := json.Marshal(payload)
		resp, err := client.Post(adminURL+"/api/routes", "application/json", bytes.NewReader(data))
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusCreated {
			fmt.Printf("Route '%s' added successfully!\n", id)
		} else {
			fmt.Printf("Failed to add route: %s\n", string(body))
		}
	},
}

var routeDeleteCmd = &cobra.Command{
	Use:	"delete [id]",
	Short:	"Delete a route by ID",
	Args:	cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		req, _ := http.NewRequest(http.MethodDelete, adminURL+"/api/routes/"+id, nil)
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusOK {
			fmt.Printf("Route '%s' deleted successfully!\n", id)
		} else {
			fmt.Printf("Failed to delete route: %s\n", string(body))
		}
	},
}

func init() {
	rootCmd.AddCommand(routeCmd)
	routeCmd.AddCommand(routeListCmd)
	routeCmd.AddCommand(routeAddCmd)
	routeCmd.AddCommand(routeDeleteCmd)

	routeAddCmd.Flags().String("id", "", "Unique route ID (required)")
	routeAddCmd.Flags().String("virtual-host", "", "Virtual host e.g. example.com (required)")
	routeAddCmd.Flags().String("upstream", "", "Target upstream pool (required)")
	routeAddCmd.Flags().String("path-prefix", "", "Path prefix match")
	routeAddCmd.MarkFlagRequired("id")
	routeAddCmd.MarkFlagRequired("virtual-host")
	routeAddCmd.MarkFlagRequired("upstream")
}
