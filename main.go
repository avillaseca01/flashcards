package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var db *sql.DB

type Card struct {
	ID          int64   `json:"id"`
	Question    string  `json:"question"`
	Answer      string  `json:"answer"`
	Category    string  `json:"category"`
	Pile        int     `json:"pile"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
	LastReview  *string `json:"last_reviewed"`
	ReviewCount int     `json:"review_count"`
}

type Stats struct {
	Total       int     `json:"total"`
	Pile1       int     `json:"pile1"`
	Pile2       int     `json:"pile2"`
	Pile3       int     `json:"pile3"`
	Leyes       int     `json:"leyes"`
	Tecnologia  int     `json:"tecnologia"`
	Progress    float64 `json:"progress_percent"`
}

func initDB() {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "/data"
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatal(err)
	}
	dbPath := filepath.Join(dataDir, "flashcards.db")
	var err error
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS cards (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			question     TEXT    NOT NULL,
			answer       TEXT    NOT NULL,
			category     TEXT    NOT NULL CHECK(category IN ('Leyes','Tecnología')),
			pile         INTEGER NOT NULL DEFAULT 1 CHECK(pile IN (1,2,3)),
			created_at   TEXT    NOT NULL,
			updated_at   TEXT    NOT NULL,
			last_reviewed TEXT,
			review_count INTEGER NOT NULL DEFAULT 0
		);
		PRAGMA journal_mode=WAL;
	`)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("DB ready: %s", dbPath)
}

func now() string { return time.Now().UTC().Format(time.RFC3339) }

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func scanCard(row interface{ Scan(...any) error }) (*Card, error) {
	c := &Card{}
	return c, row.Scan(&c.ID, &c.Question, &c.Answer, &c.Category, &c.Pile,
		&c.CreatedAt, &c.UpdatedAt, &c.LastReview, &c.ReviewCount)
}

func getCards(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	cat := q.Get("category")
	pile := q.Get("pile")
	search := q.Get("search")

	query := "SELECT id,question,answer,category,pile,created_at,updated_at,last_reviewed,review_count FROM cards WHERE 1=1"
	var args []any
	if cat != "" {
		query += " AND category=?"
		args = append(args, cat)
	}
	if pile != "" {
		query += " AND pile=?"
		args = append(args, pile)
	}
	if search != "" {
		query += " AND (question LIKE ? OR answer LIKE ? OR category LIKE ?)"
		s := "%" + search + "%"
		args = append(args, s, s, s)
	}
	query += " ORDER BY created_at ASC"

	rows, err := db.Query(query, args...)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	defer rows.Close()

	cards := make([]Card, 0)
	for rows.Next() {
		c, err := scanCard(rows)
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		cards = append(cards, *c)
	}
	writeJSON(w, 200, cards)
}

func getStats(w http.ResponseWriter, r *http.Request) {
	var s Stats
	db.QueryRow("SELECT COUNT(*) FROM cards").Scan(&s.Total)
	db.QueryRow("SELECT COUNT(*) FROM cards WHERE pile=1").Scan(&s.Pile1)
	db.QueryRow("SELECT COUNT(*) FROM cards WHERE pile=2").Scan(&s.Pile2)
	db.QueryRow("SELECT COUNT(*) FROM cards WHERE pile=3").Scan(&s.Pile3)
	db.QueryRow("SELECT COUNT(*) FROM cards WHERE category='Leyes'").Scan(&s.Leyes)
	db.QueryRow("SELECT COUNT(*) FROM cards WHERE category='Tecnología'").Scan(&s.Tecnologia)
	if s.Total > 0 {
		s.Progress = float64(s.Pile3) / float64(s.Total) * 100
		s.Progress = float64(int(s.Progress*10)) / 10
	}
	writeJSON(w, 200, s)
}

func createCard(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Question string `json:"question"`
		Answer   string `json:"answer"`
		Category string `json:"category"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeErr(w, 400, "invalid JSON")
		return
	}
	input.Question = strings.TrimSpace(input.Question)
	input.Answer = strings.TrimSpace(input.Answer)
	if input.Question == "" || input.Answer == "" {
		writeErr(w, 400, "question and answer required")
		return
	}
	if input.Category != "Leyes" && input.Category != "Tecnología" {
		writeErr(w, 400, "category must be Leyes or Tecnología")
		return
	}
	n := now()
	res, err := db.Exec(
		"INSERT INTO cards (question,answer,category,pile,created_at,updated_at) VALUES (?,?,?,1,?,?)",
		input.Question, input.Answer, input.Category, n, n,
	)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	id, _ := res.LastInsertId()
	row := db.QueryRow("SELECT id,question,answer,category,pile,created_at,updated_at,last_reviewed,review_count FROM cards WHERE id=?", id)
	c, _ := scanCard(row)
	writeJSON(w, 201, c)
}

func updateCard(w http.ResponseWriter, r *http.Request, id int64) {
	var input struct {
		Question *string `json:"question"`
		Answer   *string `json:"answer"`
		Category *string `json:"category"`
		Pile     *int    `json:"pile"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeErr(w, 400, "invalid JSON")
		return
	}
	row := db.QueryRow("SELECT id,question,answer,category,pile,created_at,updated_at,last_reviewed,review_count FROM cards WHERE id=?", id)
	c, err := scanCard(row)
	if err != nil {
		writeErr(w, 404, "not found")
		return
	}
	if input.Question != nil { c.Question = strings.TrimSpace(*input.Question) }
	if input.Answer != nil   { c.Answer = strings.TrimSpace(*input.Answer) }
	if input.Category != nil { c.Category = *input.Category }
	if input.Pile != nil     { c.Pile = *input.Pile }
	if c.Category != "Leyes" && c.Category != "Tecnología" {
		writeErr(w, 400, "invalid category")
		return
	}
	if c.Pile < 1 || c.Pile > 3 {
		writeErr(w, 400, "pile must be 1-3")
		return
	}
	_, err = db.Exec(
		"UPDATE cards SET question=?,answer=?,category=?,pile=?,updated_at=? WHERE id=?",
		c.Question, c.Answer, c.Category, c.Pile, now(), id,
	)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	row2 := db.QueryRow("SELECT id,question,answer,category,pile,created_at,updated_at,last_reviewed,review_count FROM cards WHERE id=?", id)
	updated, _ := scanCard(row2)
	writeJSON(w, 200, updated)
}

func movePile(w http.ResponseWriter, r *http.Request, id int64) {
	var input struct {
		Pile int `json:"pile"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeErr(w, 400, "invalid JSON")
		return
	}
	if input.Pile < 1 || input.Pile > 3 {
		writeErr(w, 400, "pile must be 1-3")
		return
	}
	n := now()
	res, err := db.Exec(
		"UPDATE cards SET pile=?,updated_at=?,last_reviewed=?,review_count=review_count+1 WHERE id=?",
		input.Pile, n, n, id,
	)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		writeErr(w, 404, "not found")
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func deleteCard(w http.ResponseWriter, r *http.Request, id int64) {
	res, err := db.Exec("DELETE FROM cards WHERE id=?", id)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		writeErr(w, 404, "not found")
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func deleteCards(w http.ResponseWriter, r *http.Request) {
	ids := r.URL.Query()["ids"]
	if len(ids) == 0 {
		writeErr(w, 400, "no ids provided")
		return
	}
	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	res, err := db.Exec(fmt.Sprintf("DELETE FROM cards WHERE id IN (%s)", placeholders), args...)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	n, _ := res.RowsAffected()
	writeJSON(w, 200, map[string]int64{"deleted": n})
}

func resetAll(w http.ResponseWriter, r *http.Request) {
	_, err := db.Exec("UPDATE cards SET pile=1")
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func exportJSON(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT question,answer,category,pile FROM cards ORDER BY id")
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	defer rows.Close()
	type Export struct {
		Question string `json:"question"`
		Answer   string `json:"answer"`
		Category string `json:"category"`
		Pile     int    `json:"pile"`
	}
	result := make([]Export, 0)
	for rows.Next() {
		var e Export
		rows.Scan(&e.Question, &e.Answer, &e.Category, &e.Pile)
		result = append(result, e)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=flashcards.json")
	json.NewEncoder(w).Encode(result)
}

func exportCSV(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT question,answer,category,pile FROM cards ORDER BY id")
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	defer rows.Close()
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=flashcards.csv")
	cw := csv.NewWriter(w)
	cw.Write([]string{"question", "answer", "category", "pile"})
	for rows.Next() {
		var q, a, cat string
		var p int
		rows.Scan(&q, &a, &cat, &p)
		cw.Write([]string{q, a, cat, strconv.Itoa(p)})
	}
	cw.Flush()
}

func importJSON(w http.ResponseWriter, r *http.Request) {
	var cards []struct {
		Question string `json:"question"`
		Answer   string `json:"answer"`
		Category string `json:"category"`
	}
	if err := json.NewDecoder(r.Body).Decode(&cards); err != nil {
		writeErr(w, 400, "invalid JSON")
		return
	}
	n := now()
	count := 0
	for _, c := range cards {
		c.Question = strings.TrimSpace(c.Question)
		c.Answer = strings.TrimSpace(c.Answer)
		if c.Question == "" || c.Answer == "" { continue }
		if c.Category != "Leyes" && c.Category != "Tecnología" { continue }
		db.Exec(
			"INSERT INTO cards (question,answer,category,pile,created_at,updated_at) VALUES (?,?,?,1,?,?)",
			c.Question, c.Answer, c.Category, n, n,
		)
		count++
	}
	writeJSON(w, 200, map[string]int{"imported": count})
}

func router(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" { w.WriteHeader(204); return }

	path := r.URL.Path

	// Static
	if path == "/" || path == "/index.html" {
		http.ServeFile(w, r, "/app/static/index.html")
		return
	}

	// API routes
	switch {
	case path == "/api/cards/stats" && r.Method == "GET":
		getStats(w, r)

	case path == "/api/cards" && r.Method == "GET":
		getCards(w, r)

	case path == "/api/cards" && r.Method == "POST":
		createCard(w, r)

	case path == "/api/cards" && r.Method == "DELETE":
		deleteCards(w, r)

	case path == "/api/reset" && r.Method == "POST":
		resetAll(w, r)

	case path == "/api/export/json" && r.Method == "GET":
		exportJSON(w, r)

	case path == "/api/export/csv" && r.Method == "GET":
		exportCSV(w, r)

	case path == "/api/import/json" && r.Method == "POST":
		importJSON(w, r)

	case strings.HasPrefix(path, "/api/cards/"):
		parts := strings.Split(strings.TrimPrefix(path, "/api/cards/"), "/")
		id, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			writeErr(w, 400, "invalid id")
			return
		}
		if len(parts) == 2 && parts[1] == "pile" && r.Method == "PATCH" {
			movePile(w, r, id)
		} else if len(parts) == 1 {
			switch r.Method {
			case "PUT":   updateCard(w, r, id)
			case "DELETE": deleteCard(w, r, id)
			default: writeErr(w, 405, "method not allowed")
			}
		} else {
			writeErr(w, 404, "not found")
		}

	default:
		// Serve static files (fallback)
		http.ServeFile(w, r, "/app/static/index.html")
	}
}

func main() {
	initDB()
	port := os.Getenv("PORT")
	if port == "" { port = "8080" }
	log.Printf("Flashcards listening on :%s", port)
	if err := http.ListenAndServe(":"+port, http.HandlerFunc(router)); err != nil {
		log.Fatal(err)
	}
}
