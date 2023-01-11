package cmd

import (
	"fmt"
	"os"
)

func Execute() {
	if err := Root().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
