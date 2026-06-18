package handler

import (
	"io"
	"net/http"
	"os"
)

func Fetch(w http.ResponseWriter, r *http.Request) {
	req, _ := http.NewRequest("GET", "https://blob.vercel-storage.com/", nil)
	token := os.Getenv("BLOB_WEBHOOK_PUBLIC_KEY")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		http.Error(w, "Couldn't fetch files", 500)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	io.Copy(w, resp.Body)
}
