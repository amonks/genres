package main

import (
	"context"
	"fmt"

	"github.com/amonks/genres/db"
	"github.com/amonks/genres/server"
	"github.com/amonks/genres/subcmd"
)

func serve(ctx context.Context, db *db.DB, args []string) error {
	subcmd := subcmd.New("serve", "run a web server")
	var (
		port = subcmd.Int("port", 9999, "http port")
	)
	if err := subcmd.Parse(args); err != nil {
		return fmt.Errorf("flag parsing err: %w", err)
	}

	addr := fmt.Sprintf(":%d", port)
	return server.Run(ctx, db, addr)
}
