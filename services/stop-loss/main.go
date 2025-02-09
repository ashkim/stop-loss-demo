package main

import (
	"log"
)

func main() {
	c, err := WaitDialTemporal()
	if err != nil {
		log.Fatal("failed to connect to Temporal server: ", err)
	}

	log.Println("connected")

	defer c.Close()

	go StartWorker(c)

	select {}
}
