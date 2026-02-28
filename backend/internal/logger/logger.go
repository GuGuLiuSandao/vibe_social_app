package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

var (
	logger          *log.Logger
	currentLogLevel = INFO
)

func Init() {
	logger = log.New(os.Stdout, "", log.LstdFlags)
}

func SetLogLevel(level LogLevel) {
	currentLogLevel = level
}

func formatMessage(level string, color string, format string, args ...interface{}) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	message := fmt.Sprintf(format, args...)
	return fmt.Sprintf("%s[%s] [%s] %s%s", color, timestamp, level, message, colorReset)
}

func Debug(format string, args ...interface{}) {
	if currentLogLevel <= DEBUG {
		logger.Println(formatMessage("DEBUG", colorCyan, format, args...))
	}
}

func Info(format string, args ...interface{}) {
	if currentLogLevel <= INFO {
		logger.Println(formatMessage("INFO", colorGreen, format, args...))
	}
}

func Warn(format string, args ...interface{}) {
	if currentLogLevel <= WARN {
		logger.Println(formatMessage("WARN", colorYellow, format, args...))
	}
}

func Error(format string, args ...interface{}) {
	if currentLogLevel <= ERROR {
		logger.Println(formatMessage("ERROR", colorRed, format, args...))
	}
}

func Fatal(format string, args ...interface{}) {
	if currentLogLevel <= ERROR {
		logger.Println(formatMessage("FATAL", colorRed, format, args...))
	}
	os.Exit(1)
}

func WSRequest(userID uint, msgType string, requestID int64, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	Info("[WS REQ] User=%d Type=%s ReqID=%d - %s", userID, msgType, requestID, message)
}

func WSResponse(userID uint, msgType string, requestID int64, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	Info("[WS RES] User=%d Type=%s ReqID=%d - %s", userID, msgType, requestID, message)
}

func WSPush(targetUserID uint, pushType string, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	Info("[WS PUSH] Target=%d Type=%s - %s", targetUserID, pushType, message)
}

func DBWrite(operation string, table string, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	Info("[DB WRITE] Op=%s Table=%s - %s", operation, table, message)
}

func DBRead(operation string, table string, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	Debug("[DB READ] Op=%s Table=%s - %s", operation, table, message)
}
