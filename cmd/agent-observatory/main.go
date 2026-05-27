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
	"github.com/nathanmauro/agent-observatory/internal/config"
	"github.com/nathanmauro/agent-observatory/internal/db"
	"github.com/nathanmauro/agent-observatory/internal/events"
	"github.com/nathanmauro/agent-observatory/internal/indexer"
	"github.com/nathanmauro/agent-observatory/internal/processes"
	"github.com/nathanmauro/agent-observatory/internal/sources"
	"github.com/nathanmauro/agent-observatory/internal/sources/augment"
	"github.com/nathanmauro/agent-observatory/internal/sources/claude"
	"github.com/nathanmauro/agent-observatory/internal/sources/codex"
	"github.com/nathanmauro/agent-observatory/internal/sources/cursor"
	"github.com/nathanmauro/agent-observatory/internal/watch"
	"github.com/nathanmauro/agent-observatory/internal/ws"
)

func main() {
	configPath := flag.String("config", "", "config file path (default ~/.config/agent-observatory/config.json)")
	addr := flag.String("addr", "", "listen address (overrides config)")
	dbPath := flag.String("db", "", "SQLite database path (overrides config)")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if *addr != "" {
		cfg.Addr = *addr
	}
	if *dbPath != "" {
		cfg.DBPath = *dbPath
	}

	database, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer database.Close()

	bus := events.NewBus()

	var allSources []sources.Source
	if ac, ok := cfg.Agents["claude"]; !ok || ac.Enabled {
		allSources = append(allSources, claude.NewSource())
	}
	if ac, ok := cfg.Agents["codex"]; !ok || ac.Enabled {
		allSources = append(allSources, codex.NewSource())
	}
	if ac, ok := cfg.Agents["augment"]; !ok || ac.Enabled {
		allSources = append(allSources, augment.NewSource())
	}
	if ac, ok := cfg.Agents["cursor"]; !ok || ac.Enabled {
		allSources = append(allSources, cursor.NewSource())
	}

	ix := indexer.New(database, bus, allSources)

	log.Println("running initial index…")
	if err := ix.IndexAll(context.Background()); err != nil {
		log.Printf("initial index: %v", err)
	}

	broker := ws.NewBroker(bus)
	go broker.Run()

	mon := processes.NewMonitor(bus)
	go mon.Run(context.Background())

	var allRoots []string
	var allExts []string
	extSet := make(map[string]bool)
	for _, src := range allSources {
		roots, _ := src.DiscoverRoots()
		allRoots = append(allRoots, roots...)
		for _, ext := range src.WatchExtensions() {
			if !extSet[ext] {
				extSet[ext] = true
				allExts = append(allExts, ext)
			}
		}
	}

	watcher := watch.New(ix, allExts)
	go func() {
		if err := watcher.Start(context.Background(), allRoots); err != nil {
			log.Printf("watcher: %v", err)
		}
	}()

	router := api.NewRouter(database, ix, broker, mon)
	srv := &http.Server{Addr: cfg.Addr, Handler: router}

	go func() {
		log.Printf("listening on %s", cfg.Addr)
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
