package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Izinkan semua origin (sesuaikan kebutuhan)
	},
}

// Map untuk menyimpan koneksi berdasarkan user_id
var clients = make(map[string]*websocket.Conn)
var mu sync.Mutex // Untuk menjaga concurrency

// Struktur JSON untuk menerima pesan
type MessageRequest struct {
	UserID  string `json:"user_id"` // Target penerima
	Message string `json:"message"` // Isi pesan
}

// WebSocket handler untuk user tertentu
func handleConnection(w http.ResponseWriter, r *http.Request) {
	// Ambil user_id dari query parameter
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	// Upgrade koneksi HTTP ke WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading:", err)
		return
	}
	defer conn.Close()

	// Simpan koneksi ke dalam map
	mu.Lock()
	clients[userID] = conn
	mu.Unlock()

	fmt.Printf("User %s connected\n", userID)

	// Loop untuk menangani pesan masuk
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("User disconnected:", userID)
			mu.Lock()
			delete(clients, userID)
			mu.Unlock()
			break
		}
	}
}

// API untuk mengirim pesan ke user tertentu
func sendMessage(w http.ResponseWriter, r *http.Request) {
	var req MessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Cari koneksi WebSocket berdasarkan user_id
	mu.Lock()
	conn, exists := clients[req.UserID]
	mu.Unlock()

	if !exists {
		http.Error(w, "User not connected", http.StatusNotFound)
		return
	}

	// Kirim pesan ke user tertentu
	err := conn.WriteMessage(websocket.TextMessage, []byte(req.Message))
	if err != nil {
		fmt.Println("Error sending message:", err)
		mu.Lock()
		delete(clients, req.UserID)
		mu.Unlock()
		http.Error(w, "Failed to send message", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Message sent successfully"))
}

func main() {
	http.HandleFunc("/ws", handleConnection) // WebSocket endpoint
	http.HandleFunc("/bc", sendMessage)      // API untuk kirim pesan

	fmt.Println("WebSocket server running on :8080")
	http.ListenAndServe(":8080", nil)
}
