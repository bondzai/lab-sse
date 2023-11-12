package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

var indexHTML []byte

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		htmlContent, err := os.ReadFile("../index.html")
		if err != nil {
			log.Fatal("Error reading HTML file:", err)
			return
		}
		indexHTML = htmlContent

		w.Write(indexHTML)
	})

	http.HandleFunc("/just-events", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "SSE not supported", http.StatusInternalServerError)
			return
		}

		fmt.Println("Request received for price...")

		w.Header().Set("Content-Type", "text/event-stream")

		priceCh := make(chan int)

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
	})

	http.ListenAndServe(":4400", nil)
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

	close(priceCh)

	fmt.Println("genCryptoPrice: Finished geenrating")
}

func formatServerSentEvent(event string, data any) (string, error) {
	m := map[string]any{
		"data": data,
	}

	buff := bytes.NewBuffer([]byte{})

	encoder := json.NewEncoder(buff)

	err := encoder.Encode(m)
	if err != nil {
		return "", err
	}

	sb := strings.Builder{}

	sb.WriteString(fmt.Sprintf("event: %s\n", event))
	sb.WriteString(fmt.Sprintf("data: %v\n\n", buff.String()))

	return sb.String(), nil
}
