package main

import (
	"context"
	"os"

	"charm.land/fang/v2"
)

func main() {
	if err := fang.Execute(
		context.Background(),
		rootCmd,
		fang.WithNotifySignal(os.Interrupt, os.Kill),
	); err != nil {
		os.Exit(1)
	}
}
