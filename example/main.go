package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/suapapa/go_webostv"
)

func main() {
	ctx := context.Background()

	// Discovery
	fmt.Println("Discovering TVs...")
	clients, err := webostv.Discover(ctx, false)
	if err != nil {
		log.Fatalf("discovery error: %v", err)
	}

	if len(clients) == 0 {
		fmt.Println("No TVs found.")
		return
	}

	client := clients[0]
	fmt.Printf("Connecting to %s...\n", client.URL)

	err = client.Connect()
	if err != nil {
		log.Fatalf("connect error: %v", err)
	}
	defer client.Close()

	// Registration
	store := make(map[string]string)
	statusChan, errChan := client.Register(store)

	select {
	case status := <-statusChan:
		if status == webostv.Prompted {
			fmt.Println("Please accept the connection on the TV!")
		}
		// Wait for next status
		status = <-statusChan
		if status == webostv.Registered {
			fmt.Println("Registration successful!")
			fmt.Printf("Store: %v\n", store)
		}
	case err := <-errChan:
		log.Fatalf("registration error: %v", err)
	case <-time.After(60 * time.Second):
		log.Fatal("registration timeout")
	}

	// Use controls
	media := &webostv.MediaControl{Control: webostv.Control{Client: client}}
	
	vol, err := media.GetVolume()
	if err == nil {
		fmt.Printf("Current volume: %v\n", vol["volume"])
	}

	system := &webostv.SystemControl{Control: webostv.Control{Client: client}}
	info, err := system.Info()
	if err == nil {
		fmt.Printf("System Info: %v\n", info)
	}
}
