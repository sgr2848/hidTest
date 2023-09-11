package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ebfe/scard"
	"github.com/gorilla/websocket"
)

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan []byte)

func main() {
	hexString := "3B8801000305064CAF1C80F6"

	// Convert hexadecimal string to binary data (byte slice)
	binaryData, err := hex.DecodeString(hexString)
	if err != nil {
		fmt.Println("Error decoding hexadecimal string:", err)
		return
	}

	// Convert binary data to a string (UTF-8 encoding)
	utf8String := string(binaryData)

	fmt.Println("UTF-8 String:", utf8String)

	http.HandleFunc("/", handleWebSocket)
	go startWebSocketServer()

	// Continue with the smart card listener as before
	ctx, err := scard.EstablishContext()
	if err != nil {
		log.Fatal("Failed to establish context:", err)
	}
	defer ctx.Release()

	var lastUID []byte

	// Create a goroutine to handle broadcasting UID data to WebSocket clients
	go func() {
		for {
			uid := <-broadcast
			log.Println("Sending this UID: ", uid)
			for client := range clients {
				err := client.WriteMessage(websocket.TextMessage, uid)
				if err != nil {
					log.Println("Failed to send UID to WebSocket client:", err)
					delete(clients, client)
				}
			}
		}
	}()

	for {
		// List available readers
		readers, err := ctx.ListReaders()
		if err != nil {
			log.Println("Failed to list readers:", err)
			// Wait for a while before trying again
			time.Sleep(5 * time.Second)
			continue
		}

		if len(readers) == 0 {
			log.Println("No smart card reader found. Waiting for a reader...")
			// Wait for a while before trying again
			time.Sleep(5 * time.Second)
			continue
		}

		// Connect to the first reader (if available)
		reader, err := ctx.Connect(readers[0], scard.ShareShared, scard.ProtocolAny)
		if err != nil {
			log.Println("Failed to connect to the reader:", err)
			// Wait for a while before trying again
			time.Sleep(5 * time.Second)
			continue
		}

		fmt.Println("Connected to the reader:", readers[0])

		defer reader.Disconnect(scard.LeaveCard)

		// Reader connected, start listening for card presence
		for {
			// Check for card presence
			_, err := reader.Status()
			if err == scard.ErrNoSmartcard {
				// No card present, wait and continue listening
				if lastUID != nil {
					broadcast <- lastUID // Send the last received UID
				}
				continue
			} else if err != nil {
				log.Println("Error checking card status:", err)
				// Wait for a while before trying again
				time.Sleep(5 * time.Second)
				break
			}

			// Send the proprietary Get Data command to retrieve the UID
			apduCommand := []byte{0xFF, 0xCA, 0x00, 0x00, 0x00}
			response, err := reader.Transmit(apduCommand)
			if err != nil {
				log.Println("Failed to send APDU command:", err)
				// Wait for a while before trying again
				time.Sleep(5 * time.Second)
				break
			}

			// Process the response data to extract the UID
			if len(response) >= 2 && response[len(response)-2] == 0x90 && response[len(response)-1] == 0x00 {
				uid := response[:len(response)-2]
				fmt.Printf("UID: %X\n", uid)
				broadcast <- []byte(hex.EncodeToString(uid))

			} else {
				fmt.Println("Error: Unable to retrieve UID")
			}
		}
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to upgrade connection:", err)
		return
	}

	clients[conn] = true
}

func startWebSocketServer() {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		socket, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Failed to upgrade connection:", err)
			return
		}
		defer socket.Close()

		clients[socket] = true

		for {
			uid := <-broadcast
			log.Println("Send to client:", uid)
			err := socket.WriteMessage(websocket.BinaryMessage, uid) // Send binary data as binary frame
			if err != nil {
				log.Println("Failed to send UID to WebSocket client:", err)
				delete(clients, socket)
				break
			}
		}
	})

	http.ListenAndServe(":8080", nil)
}

// func encodeBase64UID(uid []byte) string {
// 	return base64.StdEncoding.EncodeToString(uid)
// }
