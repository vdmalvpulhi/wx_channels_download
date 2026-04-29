package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version is set at build time via ldflags
	Version = "dev"
	// Commit is set at build time via ldflags
	Commit = "none"
	// Date is set at build time via ldflags
	Date = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "wx_channels_download",
	Short: "A tool to download WeChat Channels (视频号) videos",
	Long: `wx_channels_download is a CLI tool that helps you download videos
from WeChat Channels (微信视频号) by intercepting the video stream URLs.

Usage:
  Start the proxy server, configure your device to use it,
  then browse WeChat Channels to capture and download videos.`,
	SilenceUsage: true,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("wx_channels_download %s\n", Version)
		fmt.Printf("  commit: %s\n", Commit)
		fmt.Printf("  built:  %s\n", Date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
