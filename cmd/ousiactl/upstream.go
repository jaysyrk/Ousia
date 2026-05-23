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

var upstreamCmd = &cobra.Command{
	Use:	"upstream",
	Short:	"Manage upstream pools and endpoints",
}

var upstreamListCmd = &cobra.Command{
	Use:	"list",
	Short:	"List all upstream pools and their endpoints",
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := client.Get(adminURL + "/api/upstreams")
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

var upstreamAddEndpointCmd = &cobra.Command{
	Use:	"add-endpoint",
	Short:	"Add an endpoint to an upstream pool",
	Run: func(cmd *cobra.Command, args []string) {
		pool, _ := cmd.Flags().GetString("pool")
		id, _ := cmd.Flags().GetString("id")
		address, _ := cmd.Flags().GetString("address")
		weight, _ := cmd.Flags().GetInt("weight")

		payload := map[string]interface{}{
			"id":		id,
			"address":	address,
			"weight":	weight,
		}

		data, _ := json.Marshal(payload)
		resp, err := client.Post(fmt.Sprintf("%s/api/upstreams/%s/endpoints", adminURL, pool), "application/json", bytes.NewReader(data))
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusCreated {
			fmt.Printf("Endpoint '%s' added to pool '%s' successfully!\n", id, pool)
		} else {
			fmt.Printf("Failed to add endpoint: %s\n", string(body))
		}
	},
}

var upstreamRemoveEndpointCmd = &cobra.Command{
	Use:	"remove-endpoint",
	Short:	"Remove an endpoint from an upstream pool",
	Run: func(cmd *cobra.Command, args []string) {
		pool, _ := cmd.Flags().GetString("pool")
		id, _ := cmd.Flags().GetString("id")

		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/upstreams/%s/endpoints/%s", adminURL, pool, id), nil)
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusOK {
			fmt.Printf("Endpoint '%s' removed from pool '%s' successfully!\n", id, pool)
		} else {
			fmt.Printf("Failed to remove endpoint: %s\n", string(body))
		}
	},
}

func init() {
	rootCmd.AddCommand(upstreamCmd)
	upstreamCmd.AddCommand(upstreamListCmd)
	upstreamCmd.AddCommand(upstreamAddEndpointCmd)
	upstreamCmd.AddCommand(upstreamRemoveEndpointCmd)

	upstreamAddEndpointCmd.Flags().String("pool", "", "Upstream pool name (required)")
	upstreamAddEndpointCmd.Flags().String("id", "", "Endpoint ID (required)")
	upstreamAddEndpointCmd.Flags().String("address", "", "Endpoint address e.g. 127.0.0.1:8080 (required)")
	upstreamAddEndpointCmd.Flags().Int("weight", 1, "Endpoint weight")
	upstreamAddEndpointCmd.MarkFlagRequired("pool")
	upstreamAddEndpointCmd.MarkFlagRequired("id")
	upstreamAddEndpointCmd.MarkFlagRequired("address")

	upstreamRemoveEndpointCmd.Flags().String("pool", "", "Upstream pool name (required)")
	upstreamRemoveEndpointCmd.Flags().String("id", "", "Endpoint ID (required)")
	upstreamRemoveEndpointCmd.MarkFlagRequired("pool")
	upstreamRemoveEndpointCmd.MarkFlagRequired("id")
}
