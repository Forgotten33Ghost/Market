package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

type product struct {
	Id        uint32 `json:"id"`
	Available bool   `json:"available"`
	Name      string `json:"name"`
	Price     uint32 `json:"price"`
	Gramm     bool   `json:"gramm"`
	URL       string `json:"url"`
}

type createProduct struct {
	Available bool   `json:"available"`
	Name      string `json:"name"`
	Price     uint32 `json:"price"`
	Gramm     bool   `json:"gramm"`
	URL       string `json:"url"`
}
type admin struct {
	Login    string `json:"Login"`
	Password string `json:"Password"`
}

type deleteRequest struct {
	ID int `json:"id"`
}

func read(w http.ResponseWriter, r *http.Request) {
	connStr := fmt.Sprintf(
		"host=shop_database user=%s password=%s dbname=shop sslmode=disable",
		os.Getenv("DB_CLIENT_USER"),
		os.Getenv("DB_CLIENT_PASSWORD"),
	)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer db.Close()
	rows, err := db.Query("SELECT i_id, i_available, i_name, i_price, i_gramm, i_URL FROM ITEMS")
	if err != nil {
		http.Error(w, "Ошибка запроса к базе данных", http.StatusInternalServerError)
		log.Printf("Ошибка запроса: %v", err)
		return
	}
	defer rows.Close()
	products := []product{}

	for rows.Next() {
		p := product{}
		err := rows.Scan(&p.Id, &p.Available, &p.Name, &p.Price, &p.Gramm, &p.URL)
		if err != nil {
			http.Error(w, "Ошибка чтения данных", http.StatusInternalServerError)
			log.Printf("Ошибка сканирования: %v", err)
			return
		}
		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "Ошибка при обработке строк", http.StatusInternalServerError)
		log.Printf("Ошибка итерации: %v", err)
		return
	}

	if err := json.NewEncoder(w).Encode(products); err != nil {
		http.Error(w, "Ошибка сериализации данных", http.StatusInternalServerError)
		log.Printf("Ошибка маршалинга: %v", err)
		return
	}
}

func login(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	decoder := json.NewDecoder(r.Body)
	a := admin{}
	err := decoder.Decode(&a)
	if err != nil {
		http.Error(w, "Ошибка десериализации данных", http.StatusInternalServerError)
		log.Printf("Ошибка десериализации: %v", err)
	}
	connStr := fmt.Sprintf(
		"host=shop_database user=%s password=%s dbname=shop sslmode=disable",
		a.Login,
		a.Password,
	)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		http.Error(w, "Ошибка подключения к базе данных", http.StatusInternalServerError)
		log.Printf("Ошибка подключения: %v", err)
		json.NewEncoder(w).Encode(false)
		return
	}
	defer db.Close()
	json.NewEncoder(w).Encode(true)
}

func create(w http.ResponseWriter, r *http.Request) {
	connStr := fmt.Sprintf(
		"host=shop_database user=%s password=%s dbname=shop sslmode=disable",
		os.Getenv("DB_ADMIN_SHOP_USER"),
		os.Getenv("DB_ADMIN_SHOP_PASSWORD"),
	)
	item := createProduct{}
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "Неверные данные", http.StatusBadRequest)
		log.Printf("Ошибка десериализации данных: %v", err)
		return
	}
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("Ошибка подключения к базе данных: %v", err)
	}
	defer db.Close()
	_, err = db.Exec("INSERT INTO ITEMS (i_available, i_name, i_price, i_gramm, i_URL) VALUES ($1, $2, $3, $4, $5)", item.Available, item.Name, item.Price, item.Gramm, item.URL)
	if err != nil {
		http.Error(w, "Ошибка создания товара", http.StatusInternalServerError)
		log.Printf("Ошибка создания товара: %v", err)
		return
	}
}

func update(w http.ResponseWriter, r *http.Request) {
	var item product
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "Неверные данные", http.StatusBadRequest)
		log.Printf("Ошибка полученных данных: %v", err)
		return
	}

	connStr := fmt.Sprintf("host=shop_database user=%s password=%s dbname=shop sslmode=disable",
		os.Getenv("DB_ADMIN_SHOP_USER"), os.Getenv("DB_ADMIN_SHOP_PASSWORD"))

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		http.Error(w, "Ошибка подключения к БД", http.StatusInternalServerError)
		log.Printf("Ошибка подключения к БД: %v", err)
		return
	}
	defer db.Close()

	_, err = db.Exec(`UPDATE ITEMS SET i_name = $1, i_price = $2, i_gramm = $3, i_available = $4, i_URL = $5 WHERE i_id = $6`,
		item.Name, item.Price, item.Gramm, item.Available, item.URL, item.Id)

	if err != nil {
		http.Error(w, "Ошибка обновления", http.StatusInternalServerError)
		log.Printf("Ошибка обновления: %v", err)
		return
	}
}

func delete(w http.ResponseWriter, r *http.Request) {
	var req deleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверные данные", http.StatusBadRequest)
		return
	}

	connStr := fmt.Sprintf("host=shop_database user=%s password=%s dbname=shop sslmode=disable",
		os.Getenv("DB_ADMIN_SHOP_USER"), os.Getenv("DB_ADMIN_SHOP_PASSWORD"))

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		http.Error(w, "Ошибка подключения к БД", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	_, err = db.Exec("DELETE FROM ITEMS WHERE i_id = $1", req.ID)
	if err != nil {
		http.Error(w, "Ошибка удаления", http.StatusInternalServerError)
		return
	}
}

func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		h.ServeHTTP(w, r)
	})
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/read", read)

	mux.HandleFunc("/api/admin/login", login)
	mux.HandleFunc("/api/admin/create", create)
	mux.HandleFunc("/api/admin/update", update)
	mux.HandleFunc("/api/admin/delete", delete)

	fmt.Println("Server is listening on :8080...")
	http.ListenAndServe(":8080", withCORS(mux))
}
