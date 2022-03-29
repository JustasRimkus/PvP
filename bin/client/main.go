package main

import (
	"fmt"
	"time"

	"github.com/JustasRimkus/PvP/core"
)

func main() {
	client := core.NewMQTTClient()
	for i := 0; i < 10; i++ {
		token := client.Publish(core.MainTopic, 0, false, fmt.Sprintf("Message: %d.", i))
		token.Wait()
		time.Sleep(500 * time.Millisecond)
	}
}
