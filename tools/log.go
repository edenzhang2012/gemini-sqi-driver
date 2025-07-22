package tools

import (
	"log"
	"os"

	"github.com/sirupsen/logrus"
)

var (
	Logger *logrus.Logger
)

func LogInit() {
	//log config
	level := logrus.InfoLevel
	levelEnv := os.Getenv("LOG_LEVEL")
	if levelEnv != "" {
		var err error
		level, err = logrus.ParseLevel(levelEnv)
		if err != nil {
			log.Fatalf("Invalid log level: %s", levelEnv)
		}
	}

	Logger = logrus.New()
	Logger.SetOutput(os.Stdout)
	Logger.SetLevel(level)
	Logger.SetReportCaller(true)
}
