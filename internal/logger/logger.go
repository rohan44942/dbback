package logger

import (
	"io"
	"log/slog"
	"os"
)

var Log *slog.Logger

func Init(logFilePath string) {
	var writer io.Writer = os.Stdout
	if logFilePath != "" {
		logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err == nil {
			writer = logFile
		}
	}
	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	Log = slog.New(handler)
}
