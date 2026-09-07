package main

import (
	upload "devtools-presigned-upload"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

func main() {
	c := upload.NewClient()
	bucket := "devtools-assets"
	http.HandleFunc("/release", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var b upload.BuildEvent
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil || b.BuildID == "" || b.AssetKey == "" {
			http.Error(w, "build_id and asset_key are required", http.StatusBadRequest)
			return
		}
		if !strings.Contains(b.AssetKey, "/") {
			http.Error(w, "asset_key must include a path", http.StatusBadRequest)
			return
		}
		result, err := upload.PrepareRelease(c, bucket, b)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})
	log.Println("release service listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
