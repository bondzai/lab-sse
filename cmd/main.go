package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

var indexHTML []byte

func main() {
	htmlContent, err := os.ReadFile("../index.html")
	if err != nil {
		log.Fatal("Error reading HTML file:", err)
	}
	indexHTML = htmlContent

	http.HandleFunc("/", serveHTML)
	http.HandleFunc("/just-events", serveServerSentEvents)

	err = http.ListenAndServe(":4400", nil)
	if err != nil {
		log.Fatal("Server error:", err)
	}
}

func serveHTML(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write(indexHTML)
}

func serveServerSentEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	fmt.Println("Request received for price...")

	w.Header().Set("Content-Type", "text/event-stream")

	priceCh := make(chan int)

	// Close the price channel when the client disconnects
	defer close(priceCh)

	go genCryptoPrice(r.Context(), priceCh)

	for price := range priceCh {
		event, err := formatServerSentEvent("price-update", price)
		if err != nil {
			fmt.Println(err)
			break
		}

		_, err = fmt.Fprint(w, event)
		if err != nil {
			fmt.Println(err)
			break
		}

		flusher.Flush()
	}

	fmt.Println("Finished sending price updates...")
}

func genCryptoPrice(ctx context.Context, priceCh chan<- int) {
	r := rand.New(rand.NewSource(time.Now().Unix()))

	ticker := time.NewTicker(time.Second)

outerloop:
	for {
		select {
		case <-ctx.Done():
			break outerloop
		case <-ticker.C:
			p := r.Intn(100)
			priceCh <- p
		}
	}

	ticker.Stop()

	fmt.Println("genCryptoPrice: Finished generating")
}

func formatServerSentEvent(event string, data int) (string, error) {
	eventData := struct {
		Event string `json:"event"`
		Data  int    `json:"data"`
	}{
		Event: event,
		Data:  data,
	}

	eventJSON, err := json.Marshal(eventData)
	if err != nil {
		return "", err
	}

	// Build the Server-Sent Event string
	eventStr := fmt.Sprintf("event: %s\ndata: %s\n\n", event, eventJSON)

	return eventStr, nil
}
