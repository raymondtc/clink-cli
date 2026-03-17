package main

import (
	"fmt"
	"os"

	"github.com/raymondtc/clink-cli/pkg/api"
	"github.com/raymondtc/clink-cli/pkg/client"
	"github.com/raymondtc/clink-cli/pkg/renderer"
	"github.com/spf13/cobra"
)

var (
	accessID     string
	accessSecret string
	baseURL      string
	outputFormat string
)

var rootCmd = &cobra.Command{
	Use:   "clink",
	Short: "天润融通 CLI 工具",
	Long:  "天润融通呼叫中心命令行工具 - 查询通话记录、座席状态、发起呼叫等",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&accessID, "access-id", "",
		"Access ID (env: CLINK_ACCESS_ID)")
	rootCmd.PersistentFlags().StringVar(&accessSecret, "access-secret", "",
		"Access Secret (env: CLINK_ACCESS_SECRET)")
	rootCmd.PersistentFlags().StringVar(&baseURL, "base-url", "https://api-sh.clink.cn",
		"API base URL")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table",
		"Output format: table, json")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		renderer.PrintError(err)
		os.Exit(1)
	}
}

func createAPI() (*api.GeneratedAPI, error) {
	id := resolveAccessID()
	secret := resolveAccessSecret()
	url := resolveBaseURL()

	if id == "" || secret == "" {
		return nil, fmt.Errorf("access-id and access-secret are required")
	}

	config := &client.AuthConfig{
		AccessID:     id,
		AccessSecret: secret,
	}

	return api.NewGeneratedAPI(url, config)
}

func resolveAccessID() string {
	if accessID != "" {
		return accessID
	}
	if v := os.Getenv("CLINK_ACCESS_ID"); v != "" {
		return v
	}
	return os.Getenv("CLINK_ACCESS_KEY_ID")
}

func resolveAccessSecret() string {
	if accessSecret != "" {
		return accessSecret
	}
	if v := os.Getenv("CLINK_ACCESS_SECRET"); v != "" {
		return v
	}
	return os.Getenv("CLINK_SECRET")
}

func resolveBaseURL() string {
	if baseURL != "" {
		return baseURL
	}
	if v := os.Getenv("CLINK_BASE_URL"); v != "" {
		return v
	}
	return "https://api-sh.clink.cn"
}

// renderOutput renders data using the configured format
func renderOutput(data interface{}) error {
	r := renderer.New(renderer.Format(outputFormat))
	return r.Render(data)
}

// renderList renders a list result with total count
func renderList(data interface{}, total int) error {
	fmt.Printf("总计: %d 条\n\n", total)
	return renderOutput(data)
}

// deref safely dereferences a string pointer
func deref(s *string) string {
	if s == nil {
		return "-"
	}
	return *s
}

// derefInt safely dereferences an int pointer
func derefInt(i *int) string {
	if i == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *i)
}
