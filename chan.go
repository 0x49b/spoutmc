package main

import (
	"fmt"
	"time"
)

func main() {
	messageChannel := make(chan string)
	done := make(chan bool)

	go sender(messageChannel)         // Start sending messages in a separate goroutine
	go receiver(messageChannel, done) // Start receiving messages in another goroutine

	<-done // Block until done signal is received
}

func sender(ch chan string) {
	for i := 1; i <= 5; i++ {
		message := fmt.Sprintf("Message %d", i)
		ch <- message
		time.Sleep(2 * time.Second) // Simulate some work
	}
	close(ch)
}

func receiver(ch chan string, done chan bool) {
	for {
		message, more := <-ch
		if more {
			fmt.Println("Received:", message)
		} else {
			fmt.Println("Channel closed")
			done <- true
			return
		}
	}
}
