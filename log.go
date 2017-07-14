package liblog

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"

	wimark "bitbucket.org/wimarksystems/libwimark"
)

type ErrorLevel int

var DebugLevel ErrorLevel = ErrorLevel(0)
var InfoLevel ErrorLevel = ErrorLevel(1)
var WarningLevel ErrorLevel = ErrorLevel(2)
var CriticalLevel ErrorLevel = ErrorLevel(3)

func (l ErrorLevel) MarshalJSON() ([]byte, error) {
	switch l {
	case DebugLevel:
		return json.Marshal("DEBUG")
	case InfoLevel:
		return json.Marshal("INFO")
	case WarningLevel:
		return json.Marshal("WARNING")
	case CriticalLevel:
		return json.Marshal("CRITICAL")
	}
	return json.Marshal(fmt.Sprintf("LEVEL%v", l))
}

type LogMsg struct {
	Timestamp time.Time     `json:"timestamp"`
	Level     ErrorLevel    `json:"level"`
	Message   string        `json:"message"`
	Module    wimark.Module `json:"service"`
	SrcFile   string        `json:"src_file,omitempty"`
	SrcLine   int           `json:"src_line,omitempty"`
}

type Logger struct {
	module wimark.Module
	output chan LogMsg
	level  ErrorLevel
	color  bool
}

var singleLogger *Logger = nil

func printMessage(msg LogMsg, level ErrorLevel) {
	if msg.Level < level {
		return
	}
	bytestring, _ := json.Marshal(msg)
	fmt.Printf("%s\n", string(bytestring))
}

func (logger *Logger) log(level ErrorLevel, format string, values ...interface{}) {
	_, fileName, lineNumber, _ := runtime.Caller(2)
	logger.output <- LogMsg{
		Timestamp: time.Now(),
		Level:     level,
		Module:    logger.module,
		Message:   fmt.Sprintf(format, values...),
		SrcFile:   fileName,
		SrcLine:   lineNumber,
	}
}

// OBJECT

func Init(module wimark.Module) *Logger {
	var logger = new(Logger)
	logger.module = module
	logger.output = make(chan LogMsg)
	level := os.Getenv("ERRORLEVEL")
	switch level {
	case "CRITICAL":
		fallthrough
	case "3":
		logger.level = CriticalLevel
	case "WARNING":
		fallthrough
	case "2":
		logger.level = WarningLevel
	case "INFO":
		fallthrough
	case "1":
		logger.level = InfoLevel
	case "DEBUG":
		fallthrough
	case "0":
		logger.level = DebugLevel
	default:
		logger.level = InfoLevel
	}
	go func() {
		for msg := range logger.output {
			printMessage(msg, logger.level)
		}
	}()
	return logger
}

func (logger *Logger) Debug(format string, values ...interface{}) {
	logger.log(DebugLevel, format, values...)
}

func (logger *Logger) Info(format string, values ...interface{}) {
	logger.log(InfoLevel, format, values...)
}

func (logger *Logger) Warning(format string, values ...interface{}) {
	logger.log(WarningLevel, format, values...)
}

func (logger *Logger) Critical(format string, values ...interface{}) {
	logger.log(CriticalLevel, format, values...)
}

func (logger *Logger) Stop() {
	close(logger.output)
}

// SINGLETON

func InitSingle(module wimark.Module) {
	if singleLogger == nil {
		singleLogger = Init(module)
	}
}

func Debug(format string, values ...interface{}) {
	if singleLogger != nil {
		singleLogger.log(DebugLevel, format, values...)
	}
}

func Info(format string, values ...interface{}) {
	if singleLogger != nil {
		singleLogger.log(InfoLevel, format, values...)
	}
}

func Warning(format string, values ...interface{}) {
	if singleLogger != nil {
		singleLogger.log(WarningLevel, format, values...)
	}
}

func Critical(format string, values ...interface{}) {
	if singleLogger != nil {
		singleLogger.log(CriticalLevel, format, values...)
	}
}

func StopSingle() {
	if singleLogger != nil {
		singleLogger.Stop()
	}
	singleLogger = nil
}
