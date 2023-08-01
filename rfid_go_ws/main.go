package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/sstallion/go-hid"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow any origin for simplicity. You can implement proper origin checks in a real production environment.
	},
}

func processRFIDDataWithWebSocket(reader *hid.Device, conn *websocket.Conn) {
	// Continuously read RFID data
	for {
		buffer := make([]byte, 64)

		// Read RFID data from the reader
		n, err := reader.Read(buffer)
		if err != nil {
			log.Println("Failed to read RFID data:", err)
			return
		}

		// Process the received RFID data
		data := buffer[:n]

		// Send RFID data through WebSocket
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Println("Failed to send RFID data through WebSocket:", err)
			return
		}
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading connection:", err)
		return
	}
	defer conn.Close()

	// Find and open the HID device reader
	reader, err := hid.OpenFirst(0x076B, 0x502A)
	if err != nil {
		log.Println("Failed to open HID device:", err)
		return
	}
	defer reader.Close()

	// Set non-blocking mode
	err = reader.SetNonblock(true)
	if err != nil {
		log.Println("Failed to set non-blocking mode:", err)
		return
	}

	// Process RFID data when WebSocket connection is established
	processRFIDDataWithWebSocket(reader, conn)
}

func main() {
	http.HandleFunc("/ws", handleWebSocket)
	log.Println("WebSocket server started on http://localhost:8080/ws")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
