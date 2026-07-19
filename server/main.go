package main

import (
	"fmt"
	"os"

	"github.com/0xKa/equipment-checkout-system/server/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
