package server

import (
	"context"
	"net/http"

	"github.com/amonks/genres/db"
)

func Run(ctx context.Context, db *db.DB, addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, req *http.Request) {
	})

	srv := http.Server{Addr: addr, Handler: mux}

	errs := make(chan error)
	go func() { errs <- srv.ListenAndServe() }()

	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
		if err := srv.Shutdown(context.Background()); err != nil {
			return err
		}
		return <-errs
	}
}
