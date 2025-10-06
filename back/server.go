package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

type product struct {
	Id          uint32  `json:"id"`
	Available   bool    `json:"available"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	CategoryID  uint8   `json:"categoryID"`
	Category    string  `json:"category"`
	URL         string  `json:"url"`
}

type createProduct struct {
	Available   bool    `json:"available"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	CategoryID  uint8   `json:"categoryID"`
	URL         string  `json:"url"`
}

type admin struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type deleteRequest struct {
	ID int `json:"id"`
}

type createCategory struct {
	Name string `json:"name"`
}

type category struct {
	ID   uint32 `json:"id"`
	Name string `json:"name"`
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

// Получение всех товаров
func readProducts(w http.ResponseWriter, r *http.Request) {
	db, err := getDBWithUser(os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"))
	if err != nil {
		log.Printf("[ERROR] readProducts: Ошибка подключения к БД: %v", err)
		http.Error(w, "Ошибка подключения к БД", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT p.id, p.name, p.description, p.price, p.category_id, c.name as category_name, p.available,
		       pi.image_url
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.id
		LEFT JOIN product_images pi ON pi.product_id = p.id
	`)
	if err != nil {
		log.Printf("[ERROR] readProducts: Ошибка запроса к базе: %v", err)
		http.Error(w, "Ошибка запроса к базе", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	products := []product{}
	for rows.Next() {
		p := product{}
		if err := rows.Scan(&p.Id, &p.Name, &p.Description, &p.Price, &p.CategoryID, &p.Category, &p.Available, &p.URL); err != nil {
			log.Printf("[ERROR] readProducts: Ошибка чтения данных: %v", err)
			http.Error(w, "Ошибка чтения данных", http.StatusInternalServerError)
			return
		}
		products = append(products, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}

// Логин админа
func login(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var a admin
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		log.Printf("[ERROR] login: Ошибка декодирования логина: %v", err)
		http.Error(w, "Неверные данные", http.StatusBadRequest)
		return
	}

	db, err := getDBWithUser(a.Login, a.Password)
	if err != nil {
		log.Printf("[ERROR] login: Ошибка подключения: %v", err)
		http.Error(w, "Неверный логин или пароль", http.StatusUnauthorized)
		return
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Printf("[ERROR] login: Ошибка ping БД: %v", err)
		http.Error(w, "Неверный логин или пароль", http.StatusUnauthorized)
		return
	}

	token := generateToken()
	adminTokens.Lock()
	adminTokens.m[token] = a.Login
	adminTokens.Unlock()

	go func(t string) {
		time.Sleep(24 * time.Hour)
		delete(adminTokens.m, t)
	}(token)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

// Создание товара с возможностью загрузки изображения
func createProductHandler(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Парсим multipart-запрос (до 20 МБ)
	r.Body = http.MaxBytesReader(w, r.Body, 20<<20)
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		http.Error(w, "Ошибка парсинга формы", http.StatusBadRequest)
		return
	}

	// Извлекаем текстовые данные
	item := createProduct{
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
		Price:       0,
		Available:   r.FormValue("available") == "true",
		URL:         "",
	}
	fmt.Sscanf(r.FormValue("price"), "%f", &item.Price)
	var cat uint8
	fmt.Sscanf(r.FormValue("categoryID"), "%d", &cat)
	item.CategoryID = cat

	// Подключаемся к БД
	db, err := getDBWithUser(os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"))
	if err != nil {
		log.Printf("[ERROR] createProductHandler: Ошибка подключения к БД: %v", err)
		http.Error(w, "Ошибка подключения к БД", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Создаём запись о товаре
	var productID int
	err = db.QueryRow(
		`INSERT INTO products (name, description, price, category_id, available) 
         VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		item.Name, item.Description, item.Price, item.CategoryID, item.Available,
	).Scan(&productID)
	if err != nil {
		log.Printf("[ERROR] createProductHandler: Ошибка создания товара: %v", err)
		http.Error(w, "Ошибка создания товара", http.StatusInternalServerError)
		return
	}

	// Проверяем, есть ли файл
	file, _, err := r.FormFile("file")
	if err == nil {
		defer file.Close()

		// Декодируем изображение
		img, format, err := image.Decode(file)
		if err != nil {
			log.Printf("[ERROR] createProductHandler: Ошибка декодирования изображения: %v", err)
			http.Error(w, "Неверный формат изображения", http.StatusBadRequest)
			return
		}

		// Создаём папку uploads если нет
		os.MkdirAll("uploads", os.ModePerm)

		// Сохраняем файл как JPEG с именем по productID
		imagePath := fmt.Sprintf("uploads/%d.jpg", productID)
		out, err := os.Create(imagePath)
		if err != nil {
			log.Printf("[ERROR] createProductHandler: Ошибка создания файла: %v", err)
			http.Error(w, "Ошибка сохранения файла", http.StatusInternalServerError)
			return
		}
		defer out.Close()

		// Конвертируем и сохраняем как JPEG
		var opt jpeg.Options
		opt.Quality = 90
		if format != "jpeg" {
			err = jpeg.Encode(out, img, &opt)
		} else {
			err = jpeg.Encode(out, img, &opt)
		}
		if err != nil {
			log.Printf("[ERROR] createProductHandler: Ошибка записи JPEG: %v", err)
			http.Error(w, "Ошибка сохранения изображения", http.StatusInternalServerError)
			return
		}

		// Генерируем URL
		fileURL := fmt.Sprintf("http://localhost:8080/uploads/%d.jpg", productID)
		item.URL = fileURL

		// Записываем URL в таблицу product_images
		_, err = db.Exec("INSERT INTO product_images (product_id, image_url) VALUES ($1, $2)", productID, fileURL)
		if err != nil {
			log.Printf("[ERROR] createProductHandler: Ошибка записи URL в БД: %v", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":          productID,
		"name":        item.Name,
		"description": item.Description,
		"price":       item.Price,
		"categoryID":  item.CategoryID,
		"available":   item.Available,
		"url":         item.URL,
	})
}

// Обновление товара с хранением изображения только на диске
func updateProductHandler(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Парсим multipart-запрос (до 20 МБ)
	r.Body = http.MaxBytesReader(w, r.Body, 20<<20)
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		http.Error(w, "Ошибка парсинга формы", http.StatusBadRequest)
		return
	}

	// Извлекаем текстовые поля
	var item product
	fmt.Sscanf(r.FormValue("id"), "%d", &item.Id)
	item.Name = r.FormValue("name")
	item.Description = r.FormValue("description")
	fmt.Sscanf(r.FormValue("price"), "%f", &item.Price)
	var cat uint8
	fmt.Sscanf(r.FormValue("categoryID"), "%d", &cat)
	item.CategoryID = cat
	item.Available = r.FormValue("available") == "true"

	db, err := getDBWithUser(os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"))
	if err != nil {
		log.Printf("[ERROR] updateProductHandler: Ошибка подключения к БД: %v", err)
		http.Error(w, "Ошибка подключения к БД", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Обновляем товар
	_, err = db.Exec(
		`UPDATE products 
		 SET name=$1, description=$2, price=$3, category_id=$4, available=$5 
		 WHERE id=$6`,
		item.Name, item.Description, item.Price, item.CategoryID, item.Available, item.Id,
	)
	if err != nil {
		log.Printf("[ERROR] updateProductHandler: Ошибка обновления товара: %v", err)
		http.Error(w, "Ошибка обновления товара", http.StatusInternalServerError)
		return
	}

	// Проверяем, передано ли новое изображение
	file, _, err := r.FormFile("file")
	if err == nil {
		defer file.Close()

		// Удаляем старый файл, если существует
		oldPath := fmt.Sprintf("uploads/%d.jpg", item.Id)
		if _, err := os.Stat(oldPath); err == nil {
			os.Remove(oldPath)
		}

		// Декодируем новое изображение
		img, _, err := image.Decode(file)
		if err != nil {
			http.Error(w, "Неверный формат изображения", http.StatusBadRequest)
			return
		}

		// Сохраняем как JPEG
		os.MkdirAll("uploads", os.ModePerm)
		newPath := fmt.Sprintf("uploads/%d.jpg", item.Id)
		out, err := os.Create(newPath)
		if err != nil {
			http.Error(w, "Ошибка сохранения файла", http.StatusInternalServerError)
			return
		}
		defer out.Close()

		jpeg.Encode(out, img, &jpeg.Options{Quality: 90})
		item.URL = fmt.Sprintf("http://localhost:8080/%s", newPath)
	}

	// Подтягиваем название категории (если нужно)
	row := db.QueryRow("SELECT name FROM categories WHERE id=$1", item.CategoryID)
	_ = row.Scan(&item.Category)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

// Удаление товара
func deleteProductHandler(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req deleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[ERROR] deleteProductHandler: Ошибка декодирования: %v", err)
		http.Error(w, "Неверные данные", http.StatusBadRequest)
		return
	}

	db, err := getDBWithUser(os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"))
	if err != nil {
		log.Printf("[ERROR] deleteProductHandler: Ошибка подключения к БД: %v", err)
		http.Error(w, "Ошибка подключения к БД", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	_, err = db.Exec("DELETE FROM products WHERE id=$1", req.ID)
	if err != nil {
		log.Printf("[ERROR] deleteProductHandler: Ошибка удаления товара: %v", err)
		http.Error(w, "Ошибка удаления товара", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Создание категории
func createCategoryHandler(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var c createCategory
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		log.Printf("[ERROR] createCategoryHandler: Ошибка декодирования: %v", err)
		http.Error(w, "Неверные данные", http.StatusBadRequest)
		return
	}

	db, err := getDBWithUser(os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"))
	if err != nil {
		log.Printf("[ERROR] createCategoryHandler: Ошибка подключения к БД: %v", err)
		http.Error(w, "Ошибка подключения к БД", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO categories (name) VALUES ($1)", c.Name)
	if err != nil {
		log.Printf("[ERROR] createCategoryHandler: Ошибка вставки категории: %v", err)
		http.Error(w, "Ошибка создания категории", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// Удаление категории
func deleteCategoryHandler(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var data struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		log.Printf("[ERROR] deleteCategoryHandler: Ошибка декодирования: %v", err)
		http.Error(w, "Неверные данные", http.StatusBadRequest)
		return
	}

	db, err := getDBWithUser(os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"))
	if err != nil {
		log.Printf("[ERROR] deleteCategoryHandler: Ошибка подключения к БД: %v", err)
		http.Error(w, "Ошибка подключения к БД", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	_, err = db.Exec("DELETE FROM categories WHERE id = $1", data.ID)
	if err != nil {
		log.Printf("[ERROR] deleteCategoryHandler: Ошибка удаления категории: %v", err)
		http.Error(w, "Ошибка при удалении категории", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Получение категорий
func getCategories(w http.ResponseWriter, r *http.Request) {
	db, err := getDBWithUser(os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"))
	if err != nil {
		log.Printf("[ERROR] getCategories: Ошибка подключения к БД: %v", err)
		http.Error(w, "Ошибка подключения к БД", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT id, name FROM categories")
	if err != nil {
		log.Printf("[ERROR] getCategories: Ошибка запроса категорий: %v", err)
		http.Error(w, "Ошибка запроса категорий", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	categories := []category{}
	for rows.Next() {
		var c category
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			log.Printf("[ERROR] getCategories: Ошибка чтения категорий: %v", err)
			http.Error(w, "Ошибка чтения категорий", http.StatusInternalServerError)
			return
		}
		categories = append(categories, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categories)
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

	mux.HandleFunc("/api/read", readProducts)
	mux.HandleFunc("/api/admin/login", login)
	mux.HandleFunc("/api/admin/create", createProductHandler)
	mux.HandleFunc("/api/admin/update", updateProductHandler)
	mux.HandleFunc("/api/admin/delete", deleteProductHandler)
	mux.HandleFunc("/api/admin/category/create", createCategoryHandler)
	mux.HandleFunc("/api/admin/category/delete", deleteCategoryHandler)
	mux.HandleFunc("/api/categories", getCategories)
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))

	fmt.Println("Server is listening on :8080...")
	http.ListenAndServe(":8080", withCORS(mux))
}
