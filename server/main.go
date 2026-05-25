package main

import (
	"errors"
	"log"
	"manitor-server/config"
	"manitor-server/infra"
	"manitor-server/routes"
	"net/http"
)

func main() {
	listenAddr := loadConfig()

	err := infra.InitiateDatabaseConnection()
	if err != nil {
		log.Fatal("Database Connection - Failed!")
	}

	defer func() {
		err := infra.TerminateDatabaseConnection()
		if err != nil {
			log.Fatal("Database Connection Termination - Failed!")
		}
	}()

	// s := &Server{db: db}
	// go s.runMidnightReset()
	router := routes.GetRouter()
	config.WithCors(router)

	httpServer := &http.Server{
		Addr:    listenAddr,
		Handler: config.WithCors(router),
	}

	go func() {
		log.Printf("server listening on %s", listenAddr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen failed: %v", err)
		}
	}()

	waitForShutdown(httpServer)
}
