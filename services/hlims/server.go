// Package hlims implements the Home Lab Information Management System.
package hlims

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	listenAddress = flag.String("listen", "127.0.0.1:8080", "HTTP listen address")
	sqliteFile    = flag.String("sqlitedb", "", "path to the HLIMS SQLite database")
	demoData      = flag.Bool("demo", false, "seed an empty database with demonstration inventory")
)

//go:embed api/openapi.yaml
var openAPIFiles embed.FS

// Run starts the HLIMS HTTP server.
func Run() error {
	flag.Parse()
	if *sqliteFile == "" {
		return errors.New("--sqlitedb is required")
	}
	db, err := NewSQLiteDB(*sqliteFile)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() { _ = db.Close() }()
	if *demoData {
		if err := seedDemoInventory(context.Background(), db); err != nil {
			return fmt.Errorf("seed demo inventory: %w", err)
		}
	}
	server := &http.Server{
		Addr:              *listenAddress,
		Handler:           newHandler(db),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("Serving HLIMS on http://%s", *listenAddress)
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func newHandler(db *SQLiteDB) http.Handler {
	resolver := resolver{queries: db.queries}
	server := apiServer{db: db.db, queries: db.queries, resolver: resolver}
	mux := http.NewServeMux()
	registerAPIHandlers(mux, server)
	registerConsoleHandlers(mux, server)
	mux.HandleFunc("GET /openapi.yaml", serveOpenAPI)
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		serveRedirect(w, r, resolver)
	})
	return mux
}

func serveOpenAPI(w http.ResponseWriter, _ *http.Request) {
	contents, err := openAPIFiles.ReadFile("api/openapi.yaml")
	if err != nil {
		http.Error(w, "failed to load OpenAPI specification", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/yaml")
	_, _ = w.Write(contents)
}

func serveRedirect(w http.ResponseWriter, r *http.Request, resolver resolver) {
	path := strings.Trim(r.URL.Path, "/")
	if path == "" {
		http.Redirect(w, r, "/console/", http.StatusPermanentRedirect)
		return
	}

	parts := strings.Split(path, "/")
	via := r.URL.Query().Get("via")
	var (
		destination resolvedDestination
		err         error
	)
	if len(parts) >= 2 {
		if port, parseErr := strconv.Atoi(parts[1]); parseErr == nil {
			destination, err = resolver.machinePort(
				r.Context(),
				parts[0],
				port,
				via,
				r.URL.Query().Get("scheme"),
				strings.Join(parts[2:], "/"),
				r.URL.Query(),
			)
		} else if len(parts) >= 3 {
			destination, err = resolver.instance(
				r.Context(),
				parts[0],
				parts[1],
				parts[2],
				via,
				strings.Join(parts[3:], "/"),
				r.URL.Query(),
			)
		} else {
			err = errDestinationNotFound
		}
	} else {
		err = errDestinationNotFound
	}

	if err != nil {
		serveResolveError(w, err)
		return
	}
	w.Header().Set("Location", destination.URL)
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.WriteHeader(http.StatusFound)
}

func serveResolveError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errDestinationNotFound):
		http.Error(w, "destination not found", http.StatusNotFound)
	case errors.Is(err, errInvalidPort), errors.Is(err, errInvalidScheme), errors.Is(err, errInvalidVia):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		log.Printf("resolving destination: %v", err)
		http.Error(w, "failed to resolve destination", http.StatusInternalServerError)
	}
}
