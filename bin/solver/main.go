package main

import (
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/JustasRimkus/PvP/core"
	libSvm "github.com/ewalker544/libsvm-go"
	"github.com/sirupsen/logrus"
)

func main() {
	entries, err := readData("../../files/mix")
	if err != nil {
		logrus.WithError(err).Fatal("cannot read input data")
	}

	model := libSvm.NewModelFromFile("entry.model")

	var (
		correctM  int64
		correctNM int64
		malicious int64
	)

	var wg sync.WaitGroup

	batches := batchEntries(entries, 1000)

	wg.Add(len(batches))
	for _, batch := range batches {
		go func(batch []core.Entry) {
			defer wg.Done()

			for i, entry := range batch {
				if entry.Malicious {
					atomic.AddInt64(&malicious, 1)
				}

				prediction := model.Predict(entry.Floats())

				logrus.WithFields(logrus.Fields{
					"prediction": prediction,
					"current":    i,
					"total":      len(entries),
				}).Info("predicted an entry")

				if prediction > 0 {
					if entry.Malicious {
						atomic.AddInt64(&correctM, 1)
					}

					continue
				}

				if !entry.Malicious {
					atomic.AddInt64(&correctNM, 1)
				}
			}
		}(batch)
	}

	wg.Wait()

	logrus.WithFields(logrus.Fields{
		"total":             len(entries),
		"malicious":         malicious,
		"benign":            len(entries) - int(malicious),
		"percent":           float64(correctM+correctNM) / float64(len(entries)),
		"percent_malicious": float64(correctM) / float64(malicious),
		"percent_benign":    float64(correctNM) / float64(len(entries)-int(malicious)),
	}).Info("results")
}

func readData(file string) ([]core.Entry, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	return core.NewFromLines(strings.Split(string(data), "\n")...), nil
}

func batchEntries(entries []core.Entry, batchSize int) [][]core.Entry {
	var result [][]core.Entry
	for i := 0; i < len(entries); i += batchSize {
		if len(entries) > i+batchSize {
			result = append(result, entries[i:i+batchSize])
			continue
		}

		result = append(result, entries[i:])
	}

	return result
}
