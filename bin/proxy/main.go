package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/JustasRimkus/PvP/core"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
)

func main() {
	core.NewMQTTClient().Subscribe(core.MainTopic, 1, func(_ mqtt.Client, msg mqtt.Message) {
		line := msg.Payload()
		parts := strings.Split(string(line), ";")

		fields := make(logrus.Fields)
		for i, part := range parts {
			id := strconv.Itoa(i)
			if len(id) == 1 {
				id = fmt.Sprintf("0%s", id)
			}

			fields[id] = part
		}

		logrus.WithFields(fields).Info("received message")
	}).Wait()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-c:
	}
}
