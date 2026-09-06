package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"featureflags/internal/handlers"
	"featureflags/internal/middleware"
	"featureflags/internal/router"
	"featureflags/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	host := os.Getenv("HOST")
	if host == "" {
		host = "127.0.0.1"
	}

	s := store.New()
	service := handlers.New(s)
	handler := middleware.Logging(middleware.RequireAPIKey(router.New(service)))

	addr := host + ":" + port
	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	log.Printf("featureflags listening on %s", addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
