package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

type MenuItem struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Price       int    `json:"price"`
	ImageURL    string `json:"image_url"`
	Description string `json:"description"`
}

type OrderItemReq struct {
	MenuItemID int `json:"menu_item_id"`
	Quantity   int `json:"quantity"`
}

type OrderReq struct {
	TableNo int            `json:"table_no"`
	Items   []OrderItemReq `json:"items"`
}

type OrderHistoryItem struct {
	ID        int       `json:"id"`
	ItemName  string    `json:"item_name"`
	Price     int       `json:"price"`
	Quantity  int       `json:"quantity"`
	Subtotal  int       `json:"subtotal"`
	CreatedAt time.Time `json:"created_at"`
}

type BillSummary struct {
	TableNo     int                `json:"table_no"`
	TotalAmount int                `json:"total_amount"`
	TotalItems  int                `json:"total_items"`
	Details     []OrderHistoryItem `json:"details"`
}

func main() {
	var err error
	db, err = sql.Open("sqlite3", "./izakaya.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	initDB()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/menu", handleMenu)
	mux.HandleFunc("/api/orders", handleOrders)
	mux.HandleFunc("/api/orders/history", handleOrderHistory)
	mux.HandleFunc("/api/checkout", handleCheckout)

	corsMux := corsMiddleware(mux)

	log.Println("居酒屋オーダーシステム サーバーをポート8080で起動中...")
	if err := http.ListenAndServe(":8080", corsMux); err != nil {
		log.Fatal(err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Table-No, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func getTableNo(r *http.Request) int {
	tableStr := r.Header.Get("X-Table-No")
	if tableStr == "" {
		tableStr = r.URL.Query().Get("table_no")
	}
	t, err := strconv.Atoi(tableStr)
	if err != nil || t <= 0 {
		return 1
	}
	return t
}

func initDB() {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS menu_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			category TEXT,
			price INTEGER,
			image_url TEXT,
			description TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			table_no INTEGER,
			status TEXT,
			created_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS order_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			order_id INTEGER,
			menu_item_id INTEGER,
			quantity INTEGER,
			price INTEGER
		);`,
	}

	for _, stmt := range statements {
		_, err := db.Exec(stmt)
		if err != nil {
			log.Fatal(err)
		}
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM menu_items").Scan(&count)
	if count == 0 {
		db.Exec(`INSERT INTO menu_items (name, category, price, image_url, description) VALUES 
			('生ビール (中)', 'ドリンク', 550, 'https://images.unsplash.com/photo-1608270586620-248524c67de9?auto=format&fit=crop&w=400&q=80', 'キンキンに冷えた定番の生ビール'),
			('ハイボール', 'ドリンク', 480, 'https://images.unsplash.com/photo-1514362545857-3bc16c4c7d1b?auto=format&fit=crop&w=400&q=80', '爽快な炭酸とウイスキーのコク'),
			('レモンサワー', 'ドリンク', 450, 'https://images.unsplash.com/photo-1551024709-8f23befc6f87?auto=format&fit=crop&w=400&q=80', '生搾りレモンの果汁感あふれる一杯'),
			('焼き鳥 盛り合わせ (5本)', 'お料理', 880, 'https://images.unsplash.com/photo-1555939594-58d7cb561ad1?auto=format&fit=crop&w=400&q=80', 'タレ・塩おすすめの5本組み'),
			('枝豆', 'スピード', 380, 'https://images.unsplash.com/photo-1567337710282-00832b415979?auto=format&fit=crop&w=400&q=80', 'ビールにぴったりな茹でたて枝豆'),
			('だし巻き玉子', 'お料理', 620, 'https://images.unsplash.com/photo-1617093727343-374698b1b08d?auto=format&fit=crop&w=400&q=80', 'ふんわり出汁をきかせた一品'),
			('自家製ポテトサラダ', 'スピード', 420, 'https://images.unsplash.com/photo-1543339308-43e59d6b73a6?auto=format&fit=crop&w=400&q=80', 'ごろっとお芋と燻製たくあん入り'),
			('バニラアイス', 'デザート', 300, 'https://images.unsplash.com/photo-1570197788417-0e82375c9371?auto=format&fit=crop&w=400&q=80', '食後にさっぱり濃厚バニラ')`)
	}
}

func handleMenu(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rows, err := db.Query("SELECT id, name, category, price, image_url, description FROM menu_items")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var items []MenuItem
	for rows.Next() {
		var item MenuItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.Price, &item.ImageURL, &item.Description); err == nil {
			items = append(items, item)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func handleOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tableNo := getTableNo(r)

	var req OrderReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.Items) == 0 {
		http.Error(w, "注文アイテムが空です", http.StatusBadRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	res, err := tx.Exec("INSERT INTO orders (table_no, status, created_at) VALUES (?, ?, ?)", tableNo, "active", time.Now())
	if err != nil {
		tx.Rollback()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	orderID, _ := res.LastInsertId()

	for _, item := range req.Items {
		var price int
		err := tx.QueryRow("SELECT price FROM menu_items WHERE id = ?", item.MenuItemID).Scan(&price)
		if err != nil {
			tx.Rollback()
			http.Error(w, "メニューが見つかりません", http.StatusBadRequest)
			return
		}

		_, err = tx.Exec("INSERT INTO order_items (order_id, menu_item_id, quantity, price) VALUES (?, ?, ?, ?)",
			orderID, item.MenuItemID, item.Quantity, price)
		if err != nil {
			tx.Rollback()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "order_id": orderID})
}

func handleOrderHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tableNo := getTableNo(r)

	query := `
		SELECT oi.id, m.name, oi.price, oi.quantity, (oi.price * oi.quantity) as subtotal, o.created_at
		FROM order_items oi
		JOIN orders o ON oi.order_id = o.id
		JOIN menu_items m ON oi.menu_item_id = m.id
		WHERE o.table_no = ? AND o.status = 'active'
		ORDER BY o.created_at DESC
	`

	rows, err := db.Query(query, tableNo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var history []OrderHistoryItem
	for rows.Next() {
		var h OrderHistoryItem
		if err := rows.Scan(&h.ID, &h.ItemName, &h.Price, &h.Quantity, &h.Subtotal, &h.CreatedAt); err == nil {
			history = append(history, h)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

func handleCheckout(w http.ResponseWriter, r *http.Request) {
	tableNo := getTableNo(r)

	if r.Method == "GET" {
		query := `
			SELECT oi.id, m.name, oi.price, oi.quantity, (oi.price * oi.quantity) as subtotal, o.created_at
			FROM order_items oi
			JOIN orders o ON oi.order_id = o.id
			JOIN menu_items m ON oi.menu_item_id = m.id
			WHERE o.table_no = ? AND o.status = 'active'
		`
		rows, err := db.Query(query, tableNo)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var details []OrderHistoryItem
		totalAmount := 0
		totalItems := 0

		for rows.Next() {
			var h OrderHistoryItem
			if err := rows.Scan(&h.ID, &h.ItemName, &h.Price, &h.Quantity, &h.Subtotal, &h.CreatedAt); err == nil {
				details = append(details, h)
				totalAmount += h.Subtotal
				totalItems += h.Quantity
			}
		}

		summary := BillSummary{
			TableNo:     tableNo,
			TotalAmount: totalAmount,
			TotalItems:  totalItems,
			Details:     details,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(summary)

	} else if r.Method == "POST" {
		_, err := db.Exec("UPDATE orders SET status = 'completed' WHERE table_no = ? AND status = 'active'", tableNo)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "精算が完了しました。ご来店ありがとうございました。"})
	}
}