package main

import (
	"log"
	"time"

	"go.temporal.io/sdk/client"
)

func WaitDialTemporal(hostAddress string, connectionRetryAttempts int) (client.Client, error) {
	var c client.Client
	var err error

	for i := 0; i < connectionRetryAttempts; i++ {
		c, err = client.Dial(client.Options{
			HostPort: hostAddress,
		})
		if err == nil {
			log.Println("Successfully connected to Temporal server.")
			return c, nil
		}
		log.Printf("Failed to connect to Temporal (attempt %d/%d): %v. Retrying in %ds...", i+1, connectionRetryAttempts, err, 5)
		time.Sleep(5 * time.Second)
	}

	// After all retry attempts, if still failed:
	if err != nil {
		log.Printf("Failed to connect to Temporal after %d attempts.", connectionRetryAttempts)
		return nil, err // Return the last error
	}

	log.Fatal("unexpected execution")
	return c, nil
}
