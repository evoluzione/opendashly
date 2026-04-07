package main

import (
	"context"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"opendashly/backend/internal/application/bootstrap"
)

func main() {
	debug.SetMemoryLimit(400 * 1024 * 1024)
	debug.SetGCPercent(50)

	ctx := context.Background()
	app, err := bootstrap.Build(ctx)
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr:         app.ListenAddr,
		Handler:      app.Handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("listening on %s", app.ListenAddr)
	log.Fatal(srv.ListenAndServe())
}
