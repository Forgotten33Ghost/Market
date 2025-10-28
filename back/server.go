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
	"strconv"
	"strings"
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
	BuyURL      string  `json:"buyUrl"`
}

type createProduct struct {
	Available   bool    `json:"available"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	CategoryID  uint8   `json:"categoryID"`
	URL         string  `json:"url"`
	BuyURL      string  `json:"buyUrl"`
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

type productsResponse struct {
	Items []product `json:"items"`
	Total int64     `json:"total"`
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

// Получение товаров
func readProducts(w http.ResponseWriter, r *http.Request) {
	db, err := getDBWithUser(os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"))
	if err != nil {
		log.Printf("[ERROR] readProducts: Ошибка подключения к БД: %v", err)
		http.Error(w, "Ошибка подключения к БД", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	q := r.URL.Query()

	search := strings.TrimSpace(q.Get("search"))
	categoryName := strings.TrimSpace(q.Get("category"))
	categoryIDStr := strings.TrimSpace(q.Get("category_id"))
	minPriceStr := strings.TrimSpace(q.Get("min_price"))
	maxPriceStr := strings.TrimSpace(q.Get("max_price"))
	inStock := strings.EqualFold(q.Get("in_stock"), "true")

	sortParam := strings.TrimSpace(q.Get("sort"))
	orderBy := "p.price ASC"
	switch sortParam {
	case "price_asc":
		orderBy = "p.price ASC"
	case "price_desc":
		orderBy = "p.price DESC"
	case "name_asc":
		orderBy = "p.name ASC"
	case "name_desc":
		orderBy = "p.name DESC"
	}

	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 24
	}
	offset := (page - 1) * pageSize

	var where []string
	var args []any
	nextArg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	if search != "" {
		p := "%" + strings.ToLower(search) + "%"
		where = append(where, fmt.Sprintf("(LOWER(p.name) LIKE %s OR LOWER(p.description) LIKE %s)", nextArg(p), nextArg(p)))
	}
	if categoryIDStr != "" {
		where = append(where, fmt.Sprintf("p.category_id = %s", nextArg(categoryIDStr)))
	} else if categoryName != "" {
		where = append(where, fmt.Sprintf("LOWER(c.name) = %s", nextArg(strings.ToLower(categoryName))))
	}
	if minPriceStr != "" {
		where = append(where, fmt.Sprintf("p.price >= %s", nextArg(minPriceStr)))
	}
	if maxPriceStr != "" {
		where = append(where, fmt.Sprintf("p.price <= %s", nextArg(maxPriceStr)))
	}
	if inStock {
		where = append(where, "p.available = TRUE")
	}

	baseFrom := `
		FROM products p
		LEFT JOIN categories c ON c.id = p.category_id
	`
	if len(where) > 0 {
		baseFrom += " WHERE " + strings.Join(where, " AND ")
	}

	// total
	var total int64
	countSQL := "SELECT COUNT(*) " + baseFrom
	if err := db.QueryRow(countSQL, args...).Scan(&total); err != nil {
		log.Printf("[ERROR] readProducts: db count error: %v", err)
		http.Error(w, "Ошибка запроса к базе (count)", http.StatusInternalServerError)
		return
	}

	// items: первая картинка подзапросом + p.buy_url
	itemsSQL := `
  SELECT
    p.id,
    p.available,
    p.name,
    COALESCE(p.description, ''),
    p.price,
    COALESCE(p.category_id, 0) AS category_id,
    COALESCE(c.name, '') AS category,
    COALESCE((
      SELECT pi.image_url
      FROM product_images pi
      WHERE pi.product_id = p.id
      ORDER BY pi.id
      LIMIT 1
    ), '') AS url,
    COALESCE(p.buy_url, '') AS buyUrl
` + baseFrom + `
  ORDER BY ` + orderBy + `
  LIMIT $` + strconv.Itoa(len(args)+1) + ` OFFSET $` + strconv.Itoa(len(args)+2)

	argsWithPage := append([]any{}, args...)
	argsWithPage = append(argsWithPage, pageSize, offset)

	rows, err := db.Query(itemsSQL, argsWithPage...)
	if err != nil {
		log.Printf("[ERROR] readProducts: db select error: %v", err)
		http.Error(w, "Ошибка запроса к базе (select)", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := make([]product, 0, pageSize)
	for rows.Next() {
		var p product
		if err := rows.Scan(
			&p.Id,
			&p.Available,
			&p.Name,
			&p.Description,
			&p.Price,
			&p.CategoryID,
			&p.Category,
			&p.URL,
			&p.BuyURL,
		); err != nil {
			log.Printf("[ERROR] readProducts: scan error: %v", err)
			http.Error(w, "Ошибка чтения данных", http.StatusInternalServerError)
			return
		}
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[ERROR] readProducts: rows error: %v", err)
		http.Error(w, "Ошибка чтения данных", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(productsResponse{Items: items, Total: total})
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

	// до 20 МБ
	r.Body = http.MaxBytesReader(w, r.Body, 20<<20)
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		http.Error(w, "Ошибка парсинга формы", http.StatusBadRequest)
		return
	}

	item := createProduct{
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
		Available:   r.FormValue("available") == "true",
		URL:         "",
		BuyURL:      r.FormValue("buy_url"), // <— НОВОЕ ПОЛЕ
	}
	fmt.Sscanf(r.FormValue("price"), "%f", &item.Price)
	var cat uint8
	fmt.Sscanf(r.FormValue("categoryID"), "%d", &cat)
	item.CategoryID = cat

	db, err := getDBWithUser(os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"))
	if err != nil {
		log.Printf("[ERROR] createProductHandler: Ошибка подключения к БД: %v", err)
		http.Error(w, "Ошибка подключения к БД", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// сохраняем товар + buy_url
	var productID int
	err = db.QueryRow(
		`INSERT INTO products (name, description, price, category_id, available, buy_url)
         VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		item.Name, item.Description, item.Price, item.CategoryID, item.Available, item.BuyURL,
	).Scan(&productID)
	if err != nil {
		log.Printf("[ERROR] createProductHandler: Ошибка создания товара: %v", err)
		http.Error(w, "Ошибка создания товара", http.StatusInternalServerError)
		return
	}

	// изображение (необязательное)
	file, _, err := r.FormFile("file")
	if err == nil {
		defer file.Close()

		img, _, err := image.Decode(file)
		if err != nil {
			log.Printf("[ERROR] createProductHandler: Ошибка декодирования изображения: %v", err)
			http.Error(w, "Неверный формат изображения", http.StatusBadRequest)
			return
		}
		os.MkdirAll("uploads", os.ModePerm)
		imagePath := fmt.Sprintf("uploads/%d.jpg", productID)
		out, err := os.Create(imagePath)
		if err != nil {
			log.Printf("[ERROR] createProductHandler: Ошибка создания файла: %v", err)
			http.Error(w, "Ошибка сохранения файла", http.StatusInternalServerError)
			return
		}
		defer out.Close()

		if err := jpeg.Encode(out, img, &jpeg.Options{Quality: 90}); err != nil {
			log.Printf("[ERROR] createProductHandler: Ошибка записи JPEG: %v", err)
			http.Error(w, "Ошибка сохранения изображения", http.StatusInternalServerError)
			return
		}

		fileURL := fmt.Sprintf("http://localhost:8080/uploads/%d.jpg", productID)
		item.URL = fileURL
		if _, err = db.Exec("INSERT INTO product_images (product_id, image_url) VALUES ($1, $2)", productID, fileURL); err != nil {
			log.Printf("[WARN] createProductHandler: Не удалось записать URL в product_images: %v", err)
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
		"url":         item.URL,    // картинка
		"buyUrl":      item.BuyURL, // ссылка "Купить"
	})
}

// Обновление товара с хранением изображения только на диске
func updateProductHandler(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// до 20 МБ
	r.Body = http.MaxBytesReader(w, r.Body, 20<<20)
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		http.Error(w, "Ошибка парсинга формы", http.StatusBadRequest)
		return
	}

	var item product
	fmt.Sscanf(r.FormValue("id"), "%d", &item.Id)
	item.Name = r.FormValue("name")
	item.Description = r.FormValue("description")
	fmt.Sscanf(r.FormValue("price"), "%f", &item.Price)
	var cat uint8
	fmt.Sscanf(r.FormValue("categoryID"), "%d", &cat)
	item.CategoryID = cat
	item.Available = r.FormValue("available") == "true"
	item.BuyURL = r.FormValue("buy_url") // <— НОВОЕ ПОЛЕ

	db, err := getDBWithUser(os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"))
	if err != nil {
		log.Printf("[ERROR] updateProductHandler: Ошибка подключения к БД: %v", err)
		http.Error(w, "Ошибка подключения к БД", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// обновляем запись
	_, err = db.Exec(
		`UPDATE products 
		 SET name=$1, description=$2, price=$3, category_id=$4, available=$5, buy_url=$6
		 WHERE id=$7`,
		item.Name, item.Description, item.Price, item.CategoryID, item.Available, item.BuyURL, item.Id,
	)
	if err != nil {
		log.Printf("[ERROR] updateProductHandler: Ошибка обновления товара: %v", err)
		http.Error(w, "Ошибка обновления товара", http.StatusInternalServerError)
		return
	}

	// новая картинка (если прислали)
	file, _, err := r.FormFile("file")
	if err == nil {
		defer file.Close()

		// удаляем старый файл (если был)
		oldPath := fmt.Sprintf("uploads/%d.jpg", item.Id)
		if _, err := os.Stat(oldPath); err == nil {
			_ = os.Remove(oldPath)
		}

		img, _, err := image.Decode(file)
		if err != nil {
			http.Error(w, "Неверный формат изображения", http.StatusBadRequest)
			return
		}

		os.MkdirAll("uploads", os.ModePerm)
		newPath := fmt.Sprintf("uploads/%d.jpg", item.Id)
		out, err := os.Create(newPath)
		if err != nil {
			http.Error(w, "Ошибка сохранения файла", http.StatusInternalServerError)
			return
		}
		defer out.Close()

		if err := jpeg.Encode(out, img, &jpeg.Options{Quality: 90}); err != nil {
			http.Error(w, "Ошибка сохранения изображения", http.StatusInternalServerError)
			return
		}
		item.URL = fmt.Sprintf("http://localhost:8080/%s", newPath)
	}

	// подтягиваем имя категории (для удобства ответа)
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
