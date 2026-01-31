package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type WebhookPayload struct {
	Bucket        time.Time `json:"bucket"`
	PlantSourceId string    `json:"plant_source_id"`
	AvgPowerGen   *float64  `json:"avg_power_gen"`
	AvgPowerCon   *float64  `json:"avg_power_con"`
	AvgEfficiency *float64  `json:"avg_efficiency"`
	AvgTemp       *float64  `json:"avg_temp"`
	SampleCount   int       `json:"sample_count"`
	CalculatedAt  time.Time `json:"calculated_at"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9999"
	}

	http.HandleFunc("/webhook", handleWebhook)
	http.HandleFunc("/health", handleHealth)

	fmt.Printf(`
╔════════════════════════════════════════════════════════════╗
║           WEBHOOK RECEIVER - Local Testing                 ║
╠════════════════════════════════════════════════════════════╣
║  Listening on: http://localhost:%s                       ║
║  Webhook URL:  http://localhost:%s/webhook               ║
║  Health:       http://localhost:%s/health                ║
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

	// Pretty print
	prettyJSON, _ := json.MarshalIndent(payload, "  ", "  ")

	fmt.Printf(`
┌──────────────────────────────────────────────────────────────
│ 📥 WEBHOOK RECEIVED @ %s
├──────────────────────────────────────────────────────────────
│ Bucket:        %s
│ Plant ID:      %s
│ Avg Power Gen: %s MW
│ Avg Power Con: %s MW
│ Avg Efficiency: %s %%
│ Avg Temp:      %s °C
│ Sample Count:  %d
│ Calculated At: %s
├──────────────────────────────────────────────────────────────
│ Raw JSON:
%s
└──────────────────────────────────────────────────────────────

`,
		time.Now().Format("2006-01-02 15:04:05"),
		payload.Bucket.Format("2006-01-02 15:04"),
		payload.PlantSourceId,
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

func formatFloat(f *float64) string {
	if f == nil {
		return "N/A"
	}
	return fmt.Sprintf("%.2f", *f)
}
