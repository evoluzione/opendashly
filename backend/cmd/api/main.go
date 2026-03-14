package main

import (
	"context"
	"log"
	"net/http"

	"opendashly/backend/internal/bootstrap"
)

func main() {
	ctx := context.Background()
	app, err := bootstrap.Build(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("listening on %s", app.ListenAddr)
	log.Fatal(http.ListenAndServe(app.ListenAddr, app.Handler))
}
