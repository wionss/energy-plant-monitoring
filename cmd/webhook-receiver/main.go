package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

//go:embed static/*
var staticFiles embed.FS

type WebhookPayload struct {
	Bucket        time.Time `json:"bucket"`
	PlantSourceId string    `json:"plant_source_id"`
	PlantName     string    `json:"plant_name"`
	PlantType     string    `json:"plant_type"`
	Latitude      *float64  `json:"latitude"`
	Longitude     *float64  `json:"longitude"`
	AvgPowerGen   *float64  `json:"avg_power_gen"`
	AvgPowerCon   *float64  `json:"avg_power_con"`
	AvgEfficiency *float64  `json:"avg_efficiency"`
	AvgTemp       *float64  `json:"avg_temp"`
	SampleCount   int       `json:"sample_count"`
	CalculatedAt  time.Time `json:"calculated_at"`
}

// WebhookStore keeps the latest webhook payloads received per plant
type WebhookStore struct {
	mu    sync.RWMutex
	data  map[string]WebhookPayload // key: plant_source_id
	order []string                  // to preserve arrival order
}

var store = &WebhookStore{
	data:  make(map[string]WebhookPayload),
	order: make([]string, 0),
}

func (s *WebhookStore) Add(payload WebhookPayload) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Si es nueva planta, agregar al orden
	if _, exists := s.data[payload.PlantSourceId]; !exists {
		s.order = append(s.order, payload.PlantSourceId)
	}
	s.data[payload.PlantSourceId] = payload
}

func (s *WebhookStore) GetAll() []WebhookPayload {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]WebhookPayload, 0, len(s.data))
	for _, id := range s.order {
		if payload, exists := s.data[id]; exists {
			result = append(result, payload)
		}
	}
	return result
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9999"
	}

	// API endpoints
	http.HandleFunc("/webhook", handleWebhook)
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/api/plants", handleGetPlants)

	// Serve static files
	http.HandleFunc("/", handleStatic)

	fmt.Printf(`
╔════════════════════════════════════════════════════════════╗
║        WEBHOOK RECEIVER - Local Testing + Dashboard        ║
╠════════════════════════════════════════════════════════════╣
║  Dashboard:    http://localhost:%s                       ║
║  Webhook URL:  http://localhost:%s/webhook               ║
║  API Plants:   http://localhost:%s/api/plants            ║
╠════════════════════════════════════════════════════════════╣
║  Configura en .env:                                        ║
║    WEBHOOK_ENABLED=true                                    ║
║    WEBHOOK_URL=http://localhost:%s/webhook               ║
╚════════════════════════════════════════════════════════════╝

`, port, port, port, port)

	log.Printf("Webhook receiver started on port %s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func handleStatic(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}

	content, err := staticFiles.ReadFile("static" + path)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Set content type
	switch {
	case len(path) > 5 && path[len(path)-5:] == ".html":
		w.Header().Set("Content-Type", "text/html")
	case len(path) > 3 && path[len(path)-3:] == ".js":
		w.Header().Set("Content-Type", "application/javascript")
	case len(path) > 4 && path[len(path)-4:] == ".css":
		w.Header().Set("Content-Type", "text/css")
	}

	w.Write(content)
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("⚠️  [%s] Received invalid JSON: %s\n", time.Now().Format("15:04:05"), string(body))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Guardar en store
	store.Add(payload)

	// Pretty print
	prettyJSON, _ := json.MarshalIndent(payload, "  ", "  ")

	fmt.Printf(`
┌──────────────────────────────────────────────────────────────
│ 📥 WEBHOOK RECEIVED @ %s
├──────────────────────────────────────────────────────────────
│ Bucket:         %s
│ Plant ID:       %s
│ Plant Name:     %s
│ Plant Type:     %s
│ Location:       %s, %s
├──────────────────────────────────────────────────────────────
│ Avg Power Gen:  %s MW
│ Avg Power Con:  %s MW
│ Avg Efficiency: %s %%
│ Avg Temp:       %s °C
│ Sample Count:   %d
│ Calculated At:  %s
├──────────────────────────────────────────────────────────────
│ Raw JSON:
%s
└──────────────────────────────────────────────────────────────

`,
		time.Now().Format("2006-01-02 15:04:05"),
		payload.Bucket.Format("2006-01-02 15:04"),
		payload.PlantSourceId,
		payload.PlantName,
		payload.PlantType,
		formatFloat(payload.Latitude),
		formatFloat(payload.Longitude),
		formatFloat(payload.AvgPowerGen),
		formatFloat(payload.AvgPowerCon),
		formatFloat(payload.AvgEfficiency),
		formatFloat(payload.AvgTemp),
		payload.SampleCount,
		payload.CalculatedAt.Format("2006-01-02 15:04:05"),
		string(prettyJSON),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "received",
		"message": "Webhook processed successfully",
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleGetPlants(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(store.GetAll())
}

func formatFloat(f *float64) string {
	if f == nil {
		return "N/A"
	}
	return fmt.Sprintf("%.2f", *f)
}
