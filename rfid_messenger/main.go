package main

import (
	"fmt"
	"log"

	"github.com/sstallion/go-hid"
)

func main() { // Find the HID Omnikey reader
	if err := hid.Init(); err != nil {
		log.Fatal(err)
	}
	log.Println(hid.GetVersionStr())
	hid.Enumerate(hid.VendorIDAny, hid.ProductIDAny, func(info *hid.DeviceInfo) error {
		fmt.Printf("VID: %04x | PID : %04x | MFR : %s | PRODUCTSTR : %s\n",

			info.VendorID,
			info.ProductID,
			info.MfrStr,
			info.ProductStr)
		return nil
	})

	reader, err := hid.OpenFirst(0x076B, 0x502A)
	if err != nil {
		log.Fatal("Failed to open HID device:", err)
	}
	defer reader.Close()

	// Set non-blocking mode
	err = reader.SetNonblock(true)
	if err != nil {
		log.Fatal("Failed to set non-blocking mode:", err)
	}

	// Continuously read RFID data
	for {
		buffer := make([]byte, 64)

		// Read RFID data from the reader
		n, err := reader.Read(buffer)
		if err != nil {
			log.Fatal("Failed to read RFID data:", err)
		}

		// Process the received RFID data
		data := buffer[:n]
		processRFIDData(data)
	}

}
func processRFIDData(data []byte) {
	// Process the RFID data here
	fmt.Println("Received RFID data:", data)
	// Add your custom logic to handle the RFID data
}
