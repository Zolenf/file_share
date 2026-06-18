package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	mux := http.NewServeMux()

	// Serwer Go skupia się w 100% na obsłudze ruchu z formularza i bazy
	mux.HandleFunc("/api/upload", UploadHandler)
	mux.HandleFunc("/api/fetch", FetchHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
