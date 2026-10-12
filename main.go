package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

const indexHTML = `<!DOCTYPE html>
<html lang="pl">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SCI File Share</title>
    <style>
        :root { --bg-color: #1e1e2e; --surface-color: #313244; --text-color: #cdd6f4; --accent-color: #89b4fa; --border-color: #45475a; }
        body { font-family: system-ui, sans-serif; background-color: var(--bg-color); color: var(--text-color); display: flex; justify-content: center; align-items: flex-start; min-height: 100vh; margin: 0; padding: 3rem 1rem; box-sizing: border-box; }
        .container { background-color: var(--surface-color); padding: 2rem; border-radius: 12px; border: 1px solid var(--border-color); width: 100%; max-width: 600px; box-shadow: 0 10px 25px rgba(0,0,0,0.4); }
        h1 { margin-top: 0; color: var(--accent-color); text-align: center; }
        .forms-container { display: flex; gap: 1rem; flex-wrap: wrap; margin-bottom: 1.5rem; }
        .form-card { flex: 1; min-width: 250px; background: var(--bg-color); padding: 1.5rem; border-radius: 8px; border: 1px solid var(--border-color); }
        h3 { margin-top: 0; font-size: 1.1rem; color: #a6adc8; border-bottom: 1px solid var(--border-color); padding-bottom: 0.5rem; }
        form { display: flex; flex-direction: column; gap: 1rem; }
        input, textarea { padding: 0.8rem; background: var(--surface-color); border: 1px solid var(--border-color); border-radius: 6px; color: var(--text-color); width: 100%; box-sizing: border-box; }
        input[type="file"] { border: 2px dashed var(--border-color); cursor: pointer; padding: 1rem; }
        textarea { resize: vertical; min-height: 100px; }
        button { background-color: var(--accent-color); color: var(--bg-color); border: none; padding: 0.8rem; border-radius: 6px; font-weight: 600; cursor: pointer; transition: opacity 0.2s; }
        button:hover { opacity: 0.85; }
        hr { border: none; height: 1px; background-color: var(--border-color); margin: 2rem 0; }
        ul { list-style: none; padding: 0; display: flex; flex-direction: column; gap: 1rem; }
        li { background: var(--bg-color); padding: 1rem; border-radius: 8px; border: 1px solid var(--border-color); display: flex; flex-direction: column; gap: 0.5rem; }
        .file-title { font-weight: bold; font-size: 1.1rem; }
        .file-desc { font-size: 0.9rem; color: #bac2de; background: rgba(255,255,255,0.05); padding: 0.5rem; border-radius: 4px; }
        a { color: var(--accent-color); text-decoration: none; word-break: break-all; align-self: flex-start; }
        a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="container">
        <h1>SCI File Share</h1>
        
        <div class="forms-container">
            <!-- Formularz 1: Pliki -->
            <div class="form-card">
                <h3>Wyślij Plik</h3>
                <form action="/api/upload" method="POST" enctype="multipart/form-data">
                    <input type="hidden" name="type" value="file" />
                    <input type="file" name="file" required />
                    <input type="text" name="description" placeholder="Opis pliku (opcjonalnie)" />
                    <button type="submit">Wrzuć plik</button>
                </form>
            </div>

            <!-- Formularz 2: Czysty Tekst -->
            <div class="form-card">
                <h3>Zapisz Tekst</h3>
                <form action="/api/upload" method="POST" enctype="multipart/form-data">
                    <input type="hidden" name="type" value="text" />
                    <input type="text" name="textName" placeholder="Nazwa na stronie (np. notatka.txt)" required />
                    <textarea name="textContent" placeholder="Wpisz swój tekst tutaj..." required></textarea>
                    <input type="text" name="description" placeholder="Opis tekstu (opcjonalnie)" />
                    <button type="submit">Zapisz tekst</button>
                </form>
            </div>
        </div>

        <hr>
        
        <h3 style="border: none; text-align: center;">Dostępne elementy:</h3>
        <ul id="file-list"><li>Ładowanie listy...</li></ul>
    </div>
    
    <script>
        fetch('/api/fetch').then(res => res.json()).then(data => {
            const list = document.getElementById('file-list');
            if (!data.blobs || data.blobs.length === 0) { list.innerHTML = '<li>Brak plików w bazie.</li>'; return; }
            
            list.innerHTML = data.blobs.map(p => {
                let realName = p.pathname;
                let description = '';
                
                // Dekodowanie opisu zaszytego w nazwie pliku
                const separator = '---DESC---';
                if (p.pathname.includes(separator)) {
                    const parts = p.pathname.split(separator);
                    try {
                        description = decodeURIComponent(escape(atob(parts[0]))); // Bezpieczne dekodowanie UTF-8 z Base64
                    } catch(e) {
                        description = "Błąd dekodowania opisu";
                    }
                    realName = parts.slice(1).join(separator);
                }

                const descHTML = description ? '<div class="file-desc">' + description + '</div>' : '';
                return '<li>' +
                       '<span class="file-title">📄 ' + realName + '</span>' +
                       descHTML +
                       '<a href="'+p.url+'" download>Pobierz / Zobacz</a>' +
                       '</li>';
            }).join('');
        }).catch((err) => {
            document.getElementById('file-list').innerHTML = '<li>Błąd połączenia z bazą.</li>';
            console.error(err);
        });
    </script>
</body>
</html>`

func main() {
	mux := http.NewServeMux()

	// 1. Serwowanie HTML
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, indexHTML)
	})

	// 2. Logika wgrywania (Upload)
	mux.HandleFunc("/api/upload", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(10 << 20) // Limit 10MB w pamięci
		if err != nil {
			http.Error(w, "Błąd parsowania formularza", http.StatusBadRequest)
			return
		}

		uploadType := r.FormValue("type")
		description := r.FormValue("description")
		
		var fileReader io.Reader
		var filename string

		// Obsługa dwóch typów: pliku z dysku lub czystego tekstu
		if uploadType == "text" {
			textName := r.FormValue("textName")
			textContent := r.FormValue("textContent")
			
			if textName == "" || textContent == "" {
				http.Error(w, "Brakuje nazwy lub treści tekstu", http.StatusBadRequest)
				return
			}
			
			if !strings.HasSuffix(textName, ".txt") {
				textName += ".txt"
			}
			filename = textName
			fileReader = strings.NewReader(textContent)

		} else { // uploadType == "file"
			file, header, err := r.FormFile("file")
			if err != nil {
				http.Error(w, "Błąd odczytu pliku z formularza", http.StatusBadRequest)
				return
			}
			defer file.Close()
			
			filename = header.Filename
			fileReader = file
		}

		// Kodowanie opisu do Base64 (żeby uniknąć krzaków w URL Vercel Bloba)
		var finalFilename string
		if description != "" {
			encodedDesc := base64.StdEncoding.EncodeToString([]byte(description))
			finalFilename = encodedDesc + "---DESC---" + filename
		} else {
			finalFilename = filename
		}

		// Wysyłka do Vercel Blob
		blobURL := "https://blob.vercel-storage.com/" + finalFilename
		req, _ := http.NewRequest("PUT", blobURL, fileReader)
		req.Header.Set("Authorization", "Bearer "+os.Getenv("BLOB_READ_WRITE_TOKEN"))

		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != 200 {
			var body []byte
			if resp != nil {
				body, _ = io.ReadAll(resp.Body)
			}
			http.Error(w, fmt.Sprintf("Błąd Vercel API: %v, %s", err, string(body)), http.StatusInternalServerError)
			return
		}

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

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Serwer startuje na porcie %s...", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
