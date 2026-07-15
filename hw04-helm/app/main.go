package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"

	_ "github.com/lib/pq"
)

type user struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	dsn := "host=" + env("DB_HOST", "localhost") +
		" port=" + env("DB_PORT", "5432") +
		" dbname=" + env("DB_NAME", "users_db") +
		" user=" + env("DB_USER", "app") +
		" password=" + env("DB_PASSWORD", "") +
		" sslmode=disable"

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}

	writeJSON := func(w http.ResponseWriter, code int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(v)
	}
	writeErr := func(w http.ResponseWriter, code int, msg string) {
		writeJSON(w, code, map[string]string{"error": msg})
	}

	mux := http.NewServeMux()

	health := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "OK"}`))
	}
	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("GET /health/", health)

	// readiness: под не получает трафик, пока БД недоступна
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			writeErr(w, http.StatusServiceUnavailable, "database unavailable")
			return
		}
		health(w, r)
	})

	mux.HandleFunc("POST /user", func(w http.ResponseWriter, r *http.Request) {
		var u user
		if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid json")
			return
		}
		err := db.QueryRow(
			`INSERT INTO users (username, first_name, last_name, email, phone)
			 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
			u.Username, u.FirstName, u.LastName, u.Email, u.Phone).Scan(&u.ID)
		if err != nil {
			log.Println("insert:", err)
			writeErr(w, http.StatusInternalServerError, "database error")
			return
		}
		writeJSON(w, http.StatusOK, u)
	})

	mux.HandleFunc("GET /user/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid id")
			return
		}
		var u user
		err = db.QueryRow(
			`SELECT id, username, first_name, last_name, email, phone
			 FROM users WHERE id = $1`, id).
			Scan(&u.ID, &u.Username, &u.FirstName, &u.LastName, &u.Email, &u.Phone)
		if errors.Is(err, sql.ErrNoRows) {
			writeErr(w, http.StatusNotFound, "user not found")
			return
		}
		if err != nil {
			log.Println("select:", err)
			writeErr(w, http.StatusInternalServerError, "database error")
			return
		}
		writeJSON(w, http.StatusOK, u)
	})

	mux.HandleFunc("PUT /user/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid id")
			return
		}
		var u user
		if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid json")
			return
		}
		res, err := db.Exec(
			`UPDATE users SET username = $1, first_name = $2, last_name = $3, email = $4, phone = $5
			 WHERE id = $6`,
			u.Username, u.FirstName, u.LastName, u.Email, u.Phone, id)
		if err != nil {
			log.Println("update:", err)
			writeErr(w, http.StatusInternalServerError, "database error")
			return
		}
		if n, _ := res.RowsAffected(); n == 0 {
			writeErr(w, http.StatusNotFound, "user not found")
			return
		}
		u.ID = id
		writeJSON(w, http.StatusOK, u)
	})

	mux.HandleFunc("DELETE /user/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid id")
			return
		}
		res, err := db.Exec(`DELETE FROM users WHERE id = $1`, id)
		if err != nil {
			log.Println("delete:", err)
			writeErr(w, http.StatusInternalServerError, "database error")
			return
		}
		if n, _ := res.RowsAffected(); n == 0 {
			writeErr(w, http.StatusNotFound, "user not found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	log.Println("listening on :" + env("PORT", "8000"))
	log.Fatal(http.ListenAndServe(":"+env("PORT", "8000"), mux))
}
