package main

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"gopkg.in/gomail.v2"
)

type req struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Text    string `json:"text,omitempty"`
	HTML    string `json:"html,omitempty"`
	From    string `json:"from,omitempty"` // опционально, иначе из ENV
}

func main() {
	http.HandleFunc("/send", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var q req
		if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}

		host := os.Getenv("SMTP_HOST")
		port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
		user := os.Getenv("SMTP_USER")
		pass := os.Getenv("SMTP_PASS")
		from := os.Getenv("SMTP_FROM")
		if q.From != "" {
			from = q.From
		}

		m := gomail.NewMessage()
		m.SetHeader("From", from)
		m.SetHeader("To", q.To)
		m.SetHeader("Subject", q.Subject)
		if q.HTML != "" {
			m.SetBody("text/html", q.HTML)
			if q.Text != "" {
				m.AddAlternative("text/plain", q.Text)
			}
		} else {
			m.SetBody("text/plain", q.Text)
		}

		d := gomail.NewDialer(host, port, user, pass)
		d.TLSConfig = &tls.Config{
			ServerName: "mail.intbis.ru",
		}

		if err := d.DialAndSend(m); err != nil {
			http.Error(w, "send failed: "+err.Error(), http.StatusBadGateway)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	log.Println("mailer listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
