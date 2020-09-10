package main

import (
	"os"

	"github.com/yangzuo0621/gather-aks-usage/pkg"
)

func main() {
	rootCmd := pkg.CreateCommand()

	if err := rootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}
