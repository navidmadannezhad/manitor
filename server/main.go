package main

import (
	"context"
	"errors"
	"log"
	"manitor-server/config"
	"manitor-server/infra"
	"manitor-server/routes"
	"manitor-server/utils"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	defaultAddr = ":5000"
)

func waitForShutdown(httpServer *http.Server) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("shutdown signal received")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}

func main() {
	listenAddr := utils.GetFromEnv(("SERVER_CLIENT"))

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
