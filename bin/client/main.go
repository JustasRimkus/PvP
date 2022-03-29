package main

import (
	"os"
	"strings"
	"sync"
	"time"

	"github.com/JustasRimkus/PvP/core"
	"github.com/sirupsen/logrus"
)

func main() {
	malware, err := readData("../../files/malware")
	if err != nil {
		logrus.WithError(err).Fatal("cannot read malware data")
	}

	basic, err := readData("../../files/basic")
	if err != nil {
		logrus.WithError(err).Fatal("cannot read basic data")
	}

	client := core.NewMQTTClient()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < len(malware); i++ {
			client.Publish(core.MainTopic, 0, false, malware[i]).Wait()
			time.Sleep(time.Millisecond * 50)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < len(basic); i++ {
			client.Publish(core.MainTopic, 0, false, basic[i]).Wait()
			time.Sleep(time.Millisecond * 50)
		}
	}()

	wg.Wait()
}

func readData(file string) ([]string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSuffix(line, "\r")

		// Remove \r at the end of the line.
		parts := strings.Split(line, "\t")
		if len(parts) != 21 {
			continue
		}

		// Last three columns are separated by spaces.
		last := parts[20]
		parts = parts[:20]
		parts = append(parts, strings.Split(last, " ")...)

		// Add minus as a default value.
		for i, part := range parts {
			if part == "" {
				parts[i] = "-"
			}
		}

		// Rejoin the line with our own separator.
		// TODO: Create data structure, for easier management.
		lines = append(lines, strings.Join(parts, ";"))
	}

	return lines, nil
}
