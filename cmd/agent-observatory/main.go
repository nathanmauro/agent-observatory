package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nathanmauro/agent-observatory/internal/api"
	"github.com/nathanmauro/agent-observatory/internal/db"
	"github.com/nathanmauro/agent-observatory/internal/events"
	"github.com/nathanmauro/agent-observatory/internal/indexer"
	"github.com/nathanmauro/agent-observatory/internal/processes"
	"github.com/nathanmauro/agent-observatory/internal/sources/claude"
	"github.com/nathanmauro/agent-observatory/internal/watch"
	"github.com/nathanmauro/agent-observatory/internal/ws"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:3284", "listen address")
	dbPath := flag.String("db", "observatory.db", "SQLite database path")
	flag.Parse()

	database, err := db.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer database.Close()

	bus := events.NewBus()

	ix := indexer.New(database, bus)

	log.Println("running initial index…")
	if err := ix.IndexAll(context.Background()); err != nil {
		log.Printf("initial index: %v", err)
	}

	broker := ws.NewBroker(bus)
	go broker.Run()

	mon := processes.NewMonitor(bus)
	go mon.Run(context.Background())

	roots, _ := claude.DiscoverRoots("")
	watcher := watch.New(ix)
	go func() {
		if err := watcher.Start(context.Background(), roots); err != nil {
			log.Printf("watcher: %v", err)
		}
	}()

	router := api.NewRouter(database, ix, broker, mon)
	srv := &http.Server{Addr: *addr, Handler: router}

	go func() {
		log.Printf("listening on %s", *addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("serve: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("shutting down…")

	watcher.Stop()
	mon.Stop()
	broker.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
