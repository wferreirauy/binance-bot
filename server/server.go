package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/wferreirauy/binance-bot/config"
	"github.com/wferreirauy/binance-bot/storage"
)

func Serve(configFile string) error {
	var c config.Config
	cfg, err := c.Read(configFile)
	if err != nil {
		return err
	}
	addr := cfg.API.Address
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/trades", func(w http.ResponseWriter, r *http.Request) {
		limit := limitParam(r)
		records, err := storage.ReadJSONL[storage.TradeRecord](cfg.DataDir, storage.TradesFile, limit)
		writeResponse(w, records, err)
	})
	mux.HandleFunc("/api/scouts", func(w http.ResponseWriter, r *http.Request) {
		limit := limitParam(r)
		records, err := storage.ReadJSONL[storage.ScoutRecord](cfg.DataDir, storage.ScoutsFile, limit)
		writeResponse(w, records, err)
	})
	mux.HandleFunc("/api/values", func(w http.ResponseWriter, r *http.Request) {
		limit := limitParam(r)
		records, err := storage.ReadJSONL[storage.ValueRecord](cfg.DataDir, storage.ValuesFile, limit)
		writeResponse(w, records, err)
	})
	mux.HandleFunc("/api/current-asset", func(w http.ResponseWriter, r *http.Request) {
		store, err := storage.New(cfg.DataDir)
		if err != nil {
			writeError(w, err)
			return
		}
		asset, ok, err := store.CurrentAsset()
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, map[string]any{"asset": asset, "set": ok})
	})
	log.Printf("History API listening on http://%s", addr)
	return http.ListenAndServe(addr, mux)
}

func limitParam(r *http.Request) int {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	return limit
}

func writeResponse(w http.ResponseWriter, value any, err error) {
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, value)
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusInternalServerError)
}
