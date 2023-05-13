package main

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "swarmtools",
}

func main() {
	rootCmd.Execute()
}
