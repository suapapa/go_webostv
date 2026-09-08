package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/suapapa/go_webostv"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Discovery
	fmt.Println("Discovering TVs...")
	clients, err := webostv.Discover(ctx) // New API: Discover(ctx, opts...)
	if err != nil {
		log.Fatalf("discovery error: %v", err)
	}

	if len(clients) == 0 {
		fmt.Println("No TVs found.")
		return
	}

	client := clients[0]
	fmt.Printf("Connecting to %s...\n", client.URL())

	err = client.Connect()
	if err != nil {
		log.Fatalf("connect error: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("close error: %v", err)
		}
	}()

	// Registration
	store := make(map[string]string)
	// New API: Register(ctx, store)
	statusChan, errChan := client.Register(ctx, store)

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
	case <-ctx.Done():
		log.Fatal("registration timeout or cancelled")
	}

	// Use controls
	media := &webostv.MediaControl{Control: webostv.Control{Client: client}}

	// New API: methods take context
	vol, err := media.GetVolume(ctx)
	if err == nil {
		fmt.Printf("Current volume: %v (Mute: %v)\n", vol.Volume, vol.Mute)
	}

	system := &webostv.SystemControl{Control: webostv.Control{Client: client}}
	info, err := system.Info(ctx)
	if err == nil {
		fmt.Printf("System Info: %+v\n", info)
	}
}
