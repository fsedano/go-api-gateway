package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	addr := ":8081"
	if port := os.Getenv("BACKEND_PORT"); port != "" {
		addr = ":" + port
	}

	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"message":"Hello from backend!"}`)
	})

	log.Printf("starting backend on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("backend failed: %v", err)
	}
}
