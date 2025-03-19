package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Izinkan semua origin (ubah sesuai kebutuhan)
	},
}

var clients = make(map[*websocket.Conn]bool) // Simpan semua client yang terhubung

func handleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading:", err)
		return
	}
	defer conn.Close()

	clients[conn] = true
	fmt.Println("Client connected")

	// Loop untuk membaca pesan dari client (opsional)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Client disconnected:", err)
			delete(clients, conn)
			break
		}
	}
}

type MessageRequest struct {
	Message string `json:"message"`
}

func broadcastMessages(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req MessageRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Pastikan message tidak kosong
	if req.Message == "" {
		http.Error(w, "Message cannot be empty", http.StatusBadRequest)
		return
	}

	// Broadcast notifikasi ke semua WebSocket client
	broadcastNotification(req.Message)

	// Beri respon sukses
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Message broadcasted successfully"))
}

// Fungsi untuk broadcast notifikasi ke semua client
func broadcastNotification(message string) {
	for conn := range clients {
		err := conn.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			fmt.Println("Error sending message:", err)
			conn.Close()
			delete(clients, conn)
		}
	}
}

func main() {
	http.HandleFunc("/ws", handleConnection)

	http.HandleFunc("/bc", broadcastMessages)

	fmt.Println("WebSocket server running on :8080")
	http.ListenAndServe(":8080", nil)
}
