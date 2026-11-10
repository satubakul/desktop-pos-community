package main

import (
	"fmt"
	"os"

	"github.com/satubakul/desktop-pos-community/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Kasir Kita:", err)
		os.Exit(1)
	}
}
