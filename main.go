package main

import (
	"log"
	"net/http"
	"os"

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

	s := store.New()
	service := handlers.New(s)
	handler := middleware.Logging(router.New(service))

	addr := ":" + port
	log.Printf("featureflags listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}
