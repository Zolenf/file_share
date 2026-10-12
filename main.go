package main

import (
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
        
        .action-buttons { display: flex; gap: 0.5rem; margin-top: 0.5rem; }
        .btn-small { padding: 0.4rem 0.8rem; font-size: 0.9rem; border-radius: 4px; text-decoration: none; text-align: center; border: none; cursor: pointer; font-weight: 600; }
        .btn-download { background-color: var(--accent-color); color: var(--bg-color); }
        .btn-view { background-color: #f38ba8; color: #1e1e2e; }
        .btn-small:hover { opacity: 0.85; }

        /* Style dla modala (wyskakującego okna z tekstem) */
        .modal { display: none; position: fixed; z-index: 1000; left: 0; top: 0; width: 100%; height: 100%; background-color: rgba(0,0,0,0.7); justify-content: center; align-items: center; }
        .modal-content { background-color: var(--surface-color); padding: 2rem; border-radius: 12px; width: 90%; max-width: 800px; max-height: 80vh; overflow-y: auto; border: 1px solid var(--border-color); position: relative; box-shadow: 0 10px 30px rgba(0,0,0,0.6); }
        .close-btn { position: absolute; top: 1rem; right: 1.5rem; cursor: pointer; color: var(--text-color); font-size: 1.8rem; font-weight: bold; }
        .close-btn:hover { color: #f38ba8; }
        pre#modal-text { white-space: pre-wrap; word-break: break-word; color: #a6e3a1; background: var(--bg-color); padding: 1rem; border-radius: 8px; border: 1px solid var(--border-color); font-family: monospace; }
    </style>
</head>
<body>
    <div class="container">
        <h1>SCI File Share</h1>
        
        <div class="forms-container">
            <div class="form-card">
                <h3>Wyślij Plik</h3>
                <form action="/api/upload" method="POST" enctype="multipart/form-data">
                    <input type="hidden" name="type" value="file" />
                    <input type="file" name="file" required />
                    <input type="text" name="description" placeholder="Opis pliku (opcjonalnie)" />
                    <button type="submit">Wrzuć plik</button>
                </form>
            </div>

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
    
    <!-- Okno modalne do podglądu tekstu -->
    <div id="textModal" class="modal">
        <div class="modal-content">
            <span class="close-btn" onclick="closeModal()">&times;</span>
            <h3 id="modal-title" style="margin-top:0; color: var(--accent-color);">Podgląd</h3>
            <pre id="modal-text">Ładowanie...</pre>
        </div>
    </div>
    
    <script>
        // Funkcja do pokazywania modala i ładowania tekstu
        function viewText(url, name) {
            const modal = document.getElementById('textModal');
            const modalText = document.getElementById('modal-text');
            const modalTitle = document.getElementById('modal-title');
            
            modal.style.display = 'flex';
            modalTitle.innerText = name;
            modalText.innerText = 'Pobieranie zawartości...';
            
            fetch(url)
                .then(res => res.text())
                .then(text => {
                    modalText.innerText = text;
                })
                .catch(err => {
                    modalText.innerText = 'Błąd podczas ładowania pliku:\n' + err;
                });
        }

        // Zamykanie modala
        function closeModal() {
            document.getElementById('textModal').style.display = 'none';
        }

        // Zamknięcie po kliknięciu poza okienkiem
        window.onclick = function(event) {
            const modal = document.getElementById('textModal');
            if (event.target === modal) {
                modal.style.display = 'none';
            }
        }

        fetch('/api/fetch').then(res => res.json()).then(data => {
            const list = document.getElementById('file-list');
            if (!data.blobs || data.blobs.length === 0) { list.innerHTML = '<li>Brak plików w bazie.</li>'; return; }
            
            list.innerHTML = data.blobs.map(p => {
                let realName = p.pathname;
                let description = '';
                
                const separator = '---DESC---';
                if (p.pathname.includes(separator)) {
                    const parts = p.pathname.split(separator);
                    try {
                        description = decodeURIComponent(escape(atob(parts[0])));
                    } catch(e) {
                        description = "Błąd dekodowania opisu";
                    }
                    realName = parts.slice(1).join(separator);
                }

                const descHTML = description ? '<div class="file-desc">' + description + '</div>' : '';
                
                return '<li>' +
                       '<span class="file-title">📄 ' + realName + '</span>' +
                       descHTML +
                       '<div class="action-buttons">' +
                       '<a href="'+p.url+'" download class="btn-small btn-download">Pobierz</a>' +
                       '<button onclick="viewText(\''+p.url+'\', \''+realName+'\')" class="btn-small btn-view">Podgląd</button>' +
                       '</div>' +
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

		} else {
			file, header, err := r.FormFile("file")
			if err != nil {
				http.Error(w, "Błąd odczytu pliku z formularza", http.StatusBadRequest)
				return
			}
			defer file.Close()
			
			filename = header.Filename
			fileReader = file
		}

		var finalFilename string
		if description != "" {
			encodedDesc := base64.StdEncoding.EncodeToString([]byte(description))
			finalFilename = encodedDesc + "---DESC---" + filename
		} else {
			finalFilename = filename
		}

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
