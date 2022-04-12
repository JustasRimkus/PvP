package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/JustasRimkus/PvP/core"
	libSvm "github.com/ewalker544/libsvm-go"
	"github.com/sirupsen/logrus"
)

func main() {
	mix, err := readData("../../files/bo")
	if err != nil {
		logrus.WithError(err).Fatal("cannot read mix data")
	}

	output, err := os.OpenFile("entry.train", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0x644)
	if err != nil {
		logrus.WithError(err).Fatal("cannot open output file")
	}

	visited := make(map[string]struct{})
	for _, entry := range mix {
		if _, ok := visited[entry.String()]; ok {
			continue
		}

		visited[entry.String()] = struct{}{}
		if _, err := output.Write([]byte(
			fmt.Sprintf("%s\n", entry.String()),
		)); err != nil {
			logrus.WithError(err).Fatal("writing to output file")
		}
	}

	if err := output.Close(); err != nil {
		logrus.WithError(err).Error("closing output file")
	}

	param := libSvm.NewParameter()
	param.KernelType = libSvm.RBF
	param.CacheSize = 4000

	model := libSvm.NewModel(param)

	problem, err := libSvm.NewProblem("entry.train", param)
	if err != nil {
		logrus.WithError(err).Fatal("creating a problem")
	}

	model.Train(problem)
	model.Dump("../solver/entry.model")
}

func readData(file string) ([]core.Entry, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	return core.NewFromLines(strings.Split(string(data), "\n")...), nil
}
