package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/JustasRimkus/PvP/core"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
)

func main() {
	client := core.NewMQTTClient()
	token := client.Subscribe(core.MainTopic, 1, func(_ mqtt.Client, msg mqtt.Message) {
		logrus.WithField("payload", string(msg.Payload())).Info("received message")
	})
	token.Wait()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-c:
	}
}
