package api

import (
	"net/http"
	"os"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("file")

	if err != nil {
		http.Error(w, "Couldn't process the file", 400)
	}

	defer file.Close()

	blobURL := "https://blob.vercel-storage.com/" + header.Filename
	req, _ := http.NewRequest("PUT", blobURL, file)
	token := os.Getenv("BLOB_TOKEN")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		http.Error(w, "Upload do bazy zawiódł", 500)
		return
	}
	http.Error(w, "Upload Przeszedł poprawnie", 200)
	return
}
