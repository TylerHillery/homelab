// The hlims command provides CLI and TUI clients for the HLIMS API.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/TylerHillery/homelab/services/hlims/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := cli.Execute(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
