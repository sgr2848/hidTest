package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ebfe/scard"
	"github.com/gorilla/websocket"
	"github.com/sstallion/go-hid"
)

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan []byte)

func parseRFIDDataA(data []byte) {
	// Check if the length of the data buffer is three bytes
	if len(data) != 3 {
		fmt.Println("Invalid data length. Expected 3 bytes.")
		return
	}

	// Assuming the data represents a numeric value encoded in the three bytes
	value := int(data[0])*100 + int(data[1])*10 + int(data[2])

	fmt.Printf("Parsed RFID value: %d\n", value)
}
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
		// Check if the buffer exists in the mapping and is not to be ignored
		if digit, ok := digitMapping[buf]; ok && buf != [3]byte{2, 0, 0} && buf != [3]byte{2, 0, 40} {
			// Append the digit to the result
			result.WriteString(strconv.Itoa(digit))
		}
	}

	finalResult := result.String()
	broadcast <- []byte(finalResult)
	// fmt.Println("Extracted RFID number:", finalResult)
}

func hidMake() {
	err := hid.Init()
	if err != nil {
		log.Println("Error")
	}
	var rfidReaderVid uint16 = 0
	var rfidReaderPid uint16 = 0

	hid.Enumerate(hid.VendorIDAny, hid.ProductIDAny, func(info *hid.DeviceInfo) error {

		if strings.Contains(strings.ToLower(info.ProductStr), "rfid") {
			log.Println("Found RFID Reader")
			log.Println(info.BusType)
			log.Println(info.UsagePage)
			rfidReaderVid = info.VendorID
			rfidReaderPid = info.ProductID

			log.Println("PID: ", rfidReaderPid)
			log.Println("VID: ", rfidReaderVid)
		}
		return nil
	})
	log.Println()
	if rfidReaderVid == 0 && rfidReaderPid == 0 {
		log.Println("No RFID Reader Found")
	} else {
		dev, err := hid.OpenFirst(rfidReaderVid, rfidReaderPid)
		if err != nil {
			log.Println(err)
			log.Println("Failed to open")
			log.Println("Error opening device")
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
				log.Println("Error reading device:", err)
			}
			if n > 0 {
				// Convert bytes to hexadecimal string
				// hexString := hex.EncodeToString(buf[:n])
				// fmt.Println("Read Hex:", hexString)
				// broadcast <- []byte(hexString)

				log.Println("count read:", count)
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
}
func main() {
	hexString := "3B8801000305064CAF1C80F6"
	_, err := hex.DecodeString(hexString)
	if err != nil {
		fmt.Println("Error decoding hexadecimal string:", err)
		return
	}
	go hidMake()

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
			if uid == nil {
				continue
			} else {
				for client := range clients {
					time.Sleep(1 * time.Second)
					err := client.WriteMessage(websocket.TextMessage, uid)
					if err != nil {
						delete(clients, client)
					}
				}
			}
		}
	}()

	for {
		// List available readers
		readers, err := ctx.ListReaders()
		if err != nil {
			// Wait for a while before trying again
			time.Sleep(1 * time.Second)
			continue
		}

		if len(readers) == 0 {
			// broadcast <- []byte{110, 111, 95, 114, 101, 97, 100, 101, 114}

			// Wait for a while before trying again
			time.Sleep(1 * time.Second)
			continue
		}

		// Connect to the first reader (if available)
		reader, err := ctx.Connect(readers[0], scard.ShareShared, scard.ProtocolAny)
		if err != nil {
			// broadcast <- []byte{99, 111, 110, 110, 101, 99, 116, 95, 102, 97, 105, 108, 101, 100}

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
					broadcast <- lastUID // Send the last received UID
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
