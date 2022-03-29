package core

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
)

// MainTopic specifies main topic for the mqtt broker.
const MainTopic = "msg"

// NewMQTTClient creates new mqtt client with predefined default configuration.
func NewMQTTClient() mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker("tcp://127.0.0.1:1883")

	opts.OnConnect = func(client mqtt.Client) {
		logrus.Info("connected to the MQTT broker")
	}

	opts.OnConnectionLost = func(client mqtt.Client, err error) {
		logrus.WithError(err).
			Fatal("connection lost with the MQTT broker")
	}

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() {
		if err := token.Error(); err != nil {
			logrus.WithError(err).Fatal("token error")
		}
	}

	return client
}
