package main

import (
	"context"
	"log"
	"net/http"

	"opentelemetry-dashboard/backend/internal/api"
	"opentelemetry-dashboard/backend/internal/config"
	"opentelemetry-dashboard/backend/internal/query"
	"opentelemetry-dashboard/backend/internal/storage"
)

func main() {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	_, err = storage.NewClient(ctx, cfg.ClickHouseAddr)
	if err != nil {
		log.Fatal(err)
	}

	queryService := &query.Service{}
	relatedService := &query.RelatedService{}
	savedRepo := query.NewSavedQueryRepo()

	handler := api.NewRouter(queryService, relatedService, savedRepo)
	log.Printf("listening on %s", cfg.ListenAddr)
	log.Fatal(http.ListenAndServe(cfg.ListenAddr, handler))
}
