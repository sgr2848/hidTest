package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ebfe/scard"
	"github.com/gorilla/websocket"
	"github.com/sstallion/go-hid"
)

var (
	clients   = make(map[*websocket.Conn]bool)
	broadcast = make(chan []byte)
	mutex     = sync.Mutex{}
)

func parseRFIDData(buffers [][3]byte) {
	//much wow
	digitMapping := map[[3]byte]int{
		[3]byte{2, 0, 39}: 0,
		[3]byte{2, 0, 30}: 1,
		[3]byte{2, 0, 31}: 2,
		[3]byte{2, 0, 32}: 3,
		[3]byte{2, 0, 33}: 4,
		[3]byte{2, 0, 34}: 5,
		[3]byte{2, 0, 35}: 6,
		[3]byte{2, 0, 36}: 7,
		[3]byte{2, 0, 37}: 8,
		[3]byte{2, 0, 38}: 9,
	}

	var result strings.Builder

	for _, buf := range buffers {
		if digit, ok := digitMapping[buf]; ok && buf != [3]byte{2, 0, 0} && buf != [3]byte{2, 0, 40} {
			// Append the digit to the result
			result.WriteString(strconv.Itoa(digit))
		}
	}
	finalResult := result.String()
	mutex.Lock()
	broadcast <- []byte(finalResult)
	mutex.Unlock()
	// fmt.Println("Extracted RFID number:", finalResult)
}

func handleHidEvents() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered from panic:", r)
		}
	}()
	err := hid.Init()
	if err != nil {
		log.Println("Error")
	}
	var rfidReaderVid uint16 = 0
	var rfidReaderPid uint16 = 0
	for {
		hid.Enumerate(hid.VendorIDAny, hid.ProductIDAny, func(info *hid.DeviceInfo) error {

			if strings.Contains(strings.ToLower(info.ProductStr), "rfid") {
				rfidReaderVid = info.VendorID
				rfidReaderPid = info.ProductID
			}
			return nil
		})
		if rfidReaderVid == 0 && rfidReaderPid == 0 {

		} else {
			dev, err := hid.OpenFirst(rfidReaderVid, rfidReaderPid)
			if err != nil {
				return
			}
			// defer dev.Close()
			// dev.SetNonblock(true)
			count := 0
			var buffers [][3]byte
			for {
				buf := make([]byte, 3)
				n, err := dev.Read(buf)
				if err != nil {
					log.Println("Error reading:", err)
					break
				}
				if n > 0 {
					count++
					buffers = append(buffers, [3]byte{buf[0], buf[1], buf[2]})
					if len(buffers) == 26 {
						log.Println("buffers:", buffers)
						parseRFIDData(buffers)
						buffers = nil

					}
				}
			}
		}
		time.Sleep(2 * time.Second)
	}
}
func main() {
	hexString := "3B8801000305064CAF1C80F6"
	_, err := hex.DecodeString(hexString)
	if err != nil {
		fmt.Println("Error decoding hexadecimal string:", err)
		return
	}
	go handleHidEvents()
	http.HandleFunc("/", handleWebSocket)
	go startWebSocketServer()
	handleSmartCard()
}
func handleSmartCard() {
	// Continue with the smart card listener as before
	ctx, err := scard.EstablishContext()
	if err != nil {
		log.Fatal("Failed to establish context:", err)
	}
	defer ctx.Release()

	var lastUID []byte

	// Create a goroutine to handle broadcasting UID data to WebSocket clients
	// go func() {
	// 	for {
	// 		uid := <-broadcast
	// 		if uid == nil {
	// 			continue
	// 		} else {
	// 			for client := range clients {
	// 				time.Sleep(1 * time.Second)
	// 				err := client.WriteMessage(websocket.TextMessage, uid)
	// 				if err != nil {
	// 					delete(clients, client)
	// 				}
	// 			}
	// 		}
	// 	}
	// }()

	for {
		// List available readers
		readers, err := ctx.ListReaders()
		if err != nil {
			// Wait for a while before trying again
			time.Sleep(1 * time.Second)
			continue
		}

		if len(readers) == 0 {
			mutex.Lock()
			broadcast <- []byte{110, 111, 95, 114, 101, 97, 100, 101, 114}
			mutex.Unlock()
			// Wait for a while before trying again
			time.Sleep(1 * time.Second)

		}
		if len(readers) > 0 {
			// Connect to the first reader (if available)
			reader, err := ctx.Connect(readers[0], scard.ShareShared, scard.ProtocolAny)
			if err != nil {
				mutex.Lock()
				broadcast <- []byte{99, 111, 110, 110, 101, 99, 116, 95, 102, 97, 105, 108, 101, 100}
				mutex.Unlock()
				time.Sleep(2 * time.Second)
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
						mutex.Lock()
						broadcast <- lastUID // Send the last received UID
						mutex.Unlock()
					}
					continue
				} else if err != nil {

					time.Sleep(1 * time.Second)
					break
				}
				apduCommand := []byte{0xFF, 0xCA, 0x00, 0x00, 0x00}
				response, err := reader.Transmit(apduCommand)
				if err != nil {

					time.Sleep(1 * time.Second)
					break
				}

				// Process the response data to extract the UID
				if len(response) >= 2 && response[len(response)-2] == 0x90 && response[len(response)-1] == 0x00 {
					uid := response[:len(response)-2]
					mutex.Lock()
					broadcast <- []byte(hex.EncodeToString(uid))
					mutex.Unlock()

				} else {
					fmt.Println("Error: Unable to retrieve UID")
				}
			}
		}
		time.Sleep(1 * time.Second)
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade connection to WebSocket", http.StatusInternalServerError)
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
			return
		}
		defer socket.Close()

		for {
			uid := <-broadcast
			log.Println(uid)

			err := socket.WriteMessage(websocket.TextMessage, uid) // Send binary data as binary frame
			if err != nil {
				delete(clients, socket)
				break
			}
		}
	})

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("Failed to start WebSocket server:", err)
	}
}
