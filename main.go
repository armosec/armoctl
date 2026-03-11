package main

import (
	"context"
	"os"
	"syscall"

	"charm.land/fang/v2"
)

func main() {
	if err := fang.Execute(
		context.Background(),
		rootCmd,
		fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM),
	); err != nil {
		os.Exit(1)
	}
}
