package main

import (
	"context"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"opendashly/backend/internal/application/bootstrap"
	"opendashly/backend/internal/infrastructure/config"
)

func main() {
	auto := config.Auto()
	memoryLimitMiB := auto.RetentionPressureMemBudgetMB
	if memoryLimitMiB <= 0 {
		memoryLimitMiB = 400
	}
	debug.SetMemoryLimit(int64(memoryLimitMiB) * 1024 * 1024)
	debug.SetGCPercent(50)

	ctx := context.Background()
	app, err := bootstrap.Build(ctx)
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr:         app.ListenAddr,
		Handler:      app.Handler,
		ReadTimeout:  time.Duration(auto.APIReadTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(auto.APIWriteTimeoutSec) * time.Second,
		IdleTimeout:  time.Duration(auto.APIIdleTimeoutSec) * time.Second,
	}

	log.Printf("listening on %s", app.ListenAddr)
	log.Fatal(srv.ListenAndServe())
}
