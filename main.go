package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

// Cały interfejs zaszyty bezpośrednio w pamięci aplikacji
const indexHTML = `<!DOCTYPE html>
<html lang="pl">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SCI File Share</title>
    <style>
        :root { --bg-color: #1e1e2e; --surface-color: #313244; --text-color: #cdd6f4; --accent-color: #89b4fa; --border-color: #45475a; }
        body { font-family: system-ui, sans-serif; background-color: var(--bg-color); color: var(--text-color); display: flex; justify-content: center; align-items: flex-start; min-height: 100vh; margin: 0; padding: 3rem 1rem; box-sizing: border-box; }
        .container { background-color: var(--surface-color); padding: 2rem; border-radius: 12px; border: 1px solid var(--border-color); width: 100%; max-width: 500px; box-shadow: 0 10px 25px rgba(0,0,0,0.4); }
        h1 { margin-top: 0; color: var(--accent-color); text-align: center; }
        form { display: flex; flex-direction: column; gap: 1.2rem; margin-bottom: 1.5rem; }
        input[type="file"] { padding: 1rem; background: var(--bg-color); border: 2px dashed var(--border-color); border-radius: 8px; color: var(--text-color); cursor: pointer; }
        button { background-color: var(--accent-color); color: var(--bg-color); border: none; padding: 0.8rem; border-radius: 8px; font-weight: 600; cursor: pointer; }
        button:hover { opacity: 0.85; }
        hr { border: none; height: 1px; background-color: var(--border-color); margin: 2rem 0; }
        ul { list-style: none; padding: 0; display: flex; flex-direction: column; gap: 0.8rem; }
        li { background: var(--bg-color); padding: 0.8rem; border-radius: 8px; border: 1px solid var(--border-color); }
        a { color: var(--text-color); text-decoration: none; word-break: break-all; }
        a:hover { color: var(--accent-color); }
    </style>
</head>
<body>
    <div class="container">
        <h1>SCI File Share</h1>
        <form action="/api/upload" method="POST" enctype="multipart/form-data">
            <input type="file" name="file" required />
            <button type="submit">Wrzuć plik</button>
        </form>
        <hr>
        <h3>Dostępne pliki:</h3>
        <ul id="file-list"><li>Ładowanie listy...</li></ul>
    </div>
    <script>
        fetch('/api/fetch').then(res => res.json()).then(data => {
            const list = document.getElementById('file-list');
            if (!data.blobs || data.blobs.length === 0) { list.innerHTML = '<li>Brak plików w bazie.</li>'; return; }
            list.innerHTML = data.blobs.map(p => '<li>📄 <a href="'+p.url+'" download>'+p.pathname+'</a></li>').join('');
        }).catch(() => {
            document.getElementById('file-list').innerHTML = '<li>Błąd połączenia z bazą.</li>';
        });
    </script>
</body>
</html>`

func main() {
	mux := http.NewServeMux()

	// 1. Serwowanie HTML z pamięci RAM (Zero problemów z plikami statycznymi)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, indexHTML)
	})

	// 2. Logika wgrywania (Upload)
	mux.HandleFunc("/api/upload", func(w http.ResponseWriter, r *http.Request) {
		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Błąd odczytu pliku z formularza", http.StatusBadRequest)
			return
		}
		defer file.Close()

		blobURL := "https://blob.vercel-storage.com/" + header.Filename
		req, _ := http.NewRequest("PUT", blobURL, file)
		req.Header.Set("Authorization", "Bearer "+os.Getenv("BLOB_READ_WRITE_TOKEN"))

		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != 200 {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Powrót na stronę główną po sukcesie
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	// 3. Logika pobierania listy (Fetch)
	mux.HandleFunc("/api/fetch", func(w http.ResponseWriter, r *http.Request) {
		req, _ := http.NewRequest("GET", "https://blob.vercel-storage.com/", nil)
		req.Header.Set("Authorization", "Bearer "+os.Getenv("BLOB_READ_WRITE_TOKEN"))

		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != 200 {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		w.Header().Set("Content-Type", "application/json")
		io.Copy(w, resp.Body)
	})

	// Odpalenie serwera pod dyktando Vercela
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Serwer startuje na porcie %s...", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
