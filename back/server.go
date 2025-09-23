package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

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
	Login    string `json:"login"`
	Password string `json:"password"`
}

type deleteRequest struct {
	ID int `json:"id"`
}

// Хранилище токенов в памяти
var adminTokens = struct {
	sync.RWMutex
	m map[string]string
}{m: make(map[string]string)}

// Генерация случайного токена
func generateToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Проверка токена для защищённых эндпоинтов
func isAdmin(r *http.Request) bool {
	token := r.Header.Get("X-Admin-Token")
	adminTokens.RLock()
	defer adminTokens.RUnlock()
	_, ok := adminTokens.m[token]
	return ok
}

// Функция для подключения к БД от имени определённого пользователя
func getDBWithUser(user, password string) (*sql.DB, error) {
	connStr := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		user,
		password,
		os.Getenv("DB_NAME"),
	)
	return sql.Open("postgres", connStr)
}

// Публичный эндпоинт получения товаров
func read(w http.ResponseWriter, r *http.Request) {
	db, err := getDBWithUser(os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"))
	if err != nil {
		http.Error(w, "Ошибка подключения к базе данных", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT i_id, i_available, i_name, i_price, i_gramm, i_URL FROM ITEMS")
	if err != nil {
		http.Error(w, "Ошибка запроса к базе", http.StatusInternalServerError)
		log.Printf("Ошибка запроса: %v", err)
		return
	}
	defer rows.Close()

	products := []product{}
	for rows.Next() {
		p := product{}
		if err := rows.Scan(&p.Id, &p.Available, &p.Name, &p.Price, &p.Gramm, &p.URL); err != nil {
			http.Error(w, "Ошибка чтения данных", http.StatusInternalServerError)
			return
		}
		products = append(products, p)
	}

	json.NewEncoder(w).Encode(products)
}

// Эндпоинт логина админа
func login(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var a admin
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, "Неверные данные", http.StatusBadRequest)
		return
	}

	db, err := getDBWithUser(a.Login, a.Password)
	if err != nil {
		http.Error(w, "Неверный логин или пароль", http.StatusUnauthorized)
		log.Printf("Ошибка подключения: %v", err)
		return
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		http.Error(w, "Неверный логин или пароль", http.StatusUnauthorized)
		return
	}

	// Генерация токена и сохранение
	token := generateToken()
	adminTokens.Lock()
	adminTokens.m[token] = a.Login
	adminTokens.Unlock()

	go func(t string) {
		time.Sleep(24 * time.Hour)
		adminTokens.Lock()
		delete(adminTokens.m, t)
		adminTokens.Unlock()
	}(token)

	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

// Создание товара
func create(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var item createProduct
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "Неверные данные", http.StatusBadRequest)
		return
	}

	db, err := getDBWithUser(os.Getenv("DB_ADMIN_USER"), os.Getenv("DB_ADMIN_PASSWORD"))
	if err != nil {
		http.Error(w, "Ошибка подключения к БД", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO ITEMS (i_available, i_name, i_price, i_gramm, i_URL) VALUES ($1,$2,$3,$4,$5)",
		item.Available, item.Name, item.Price, item.Gramm, item.URL)
	if err != nil {
		http.Error(w, "Ошибка создания товара", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// Обновление товара
func update(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var item product
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "Неверные данные", http.StatusBadRequest)
		return
	}

	db, err := getDBWithUser(os.Getenv("DB_ADMIN_USER"), os.Getenv("DB_ADMIN_PASSWORD"))
	if err != nil {
		http.Error(w, "Ошибка подключения к БД", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	_, err = db.Exec(`UPDATE ITEMS SET i_name=$1, i_price=$2, i_gramm=$3, i_available=$4, i_URL=$5 WHERE i_id=$6`,
		item.Name, item.Price, item.Gramm, item.Available, item.URL, item.Id)
	if err != nil {
		http.Error(w, "Ошибка обновления", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Удаление товара
func delete(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req deleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверные данные", http.StatusBadRequest)
		return
	}

	db, err := getDBWithUser(os.Getenv("DB_ADMIN_USER"), os.Getenv("DB_ADMIN_PASSWORD"))
	if err != nil {
		http.Error(w, "Ошибка подключения к БД", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	_, err = db.Exec("DELETE FROM ITEMS WHERE i_id=$1", req.ID)
	if err != nil {
		http.Error(w, "Ошибка удаления", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// CORS
func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Admin-Token")
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
