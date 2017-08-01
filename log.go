package liblog

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"time"

	wimark "bitbucket.org/wimarksystems/libwimark"
)

type LogLevel int

var DebugLevel LogLevel = LogLevel(0)
var InfoLevel LogLevel = LogLevel(1)
var WarningLevel LogLevel = LogLevel(2)
var ErrorLevel LogLevel = LogLevel(3)

func (l LogLevel) MarshalJSON() ([]byte, error) {
	switch l {
	case DebugLevel:
		return json.Marshal("DEBUG")
	case InfoLevel:
		return json.Marshal("INFO")
	case WarningLevel:
		return json.Marshal("WARNING")
	case ErrorLevel:
		return json.Marshal("ERROR")
	}
	return json.Marshal(fmt.Sprintf("LEVEL%v", l))
}

type LogMsg struct {
	Timestamp time.Time     `json:"timestamp"`
	Level     LogLevel      `json:"level"`
	Message   string        `json:"message"`
	Module    wimark.Module `json:"service"`
	SrcFile   string        `json:"src_file,omitempty"`
	SrcLine   int           `json:"src_line,omitempty"`
}

type Logger struct {
	module wimark.Module
	output chan LogMsg
	level  LogLevel
}

var singleLogger *Logger = nil

func printMessage(msg LogMsg, level LogLevel) {
	if msg.Level < level {
		return
	}
	bytestring, _ := json.Marshal(msg)
	fmt.Printf("%s\n", string(bytestring))
}

func (logger *Logger) log(level LogLevel, format string, values ...interface{}) {
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
	level := os.Getenv("LOGLEVEL")
	switch level {
	case "ERROR":
		fallthrough
	case "3":
		logger.level = ErrorLevel
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

func (logger *Logger) Error(format string, values ...interface{}) {
	logger.log(ErrorLevel, format, values...)
}

func (logger *Logger) Stop() {
	close(logger.output)
}

type LogWriter struct {
	host  *Logger
	level LogLevel
}

func (writer *LogWriter) Write(p []byte) (n int, err error) {
	msg := string(p)
	writer.host.log(writer.level, msg)
	return len(p), nil
}

func (logger *Logger) DebugWriter() io.Writer {
	return &LogWriter{logger, DebugLevel}
}
func (logger *Logger) InfoWriter() io.Writer {
	return &LogWriter{logger, InfoLevel}
}
func (logger *Logger) WarningWriter() io.Writer {
	return &LogWriter{logger, WarningLevel}
}
func (logger *Logger) ErrorWriter() io.Writer {
	return &LogWriter{logger, ErrorLevel}
}
func (logger *Logger) DebugLogger() *log.Logger {
	return log.New(logger.DebugWriter(), "", 0)
}
func (logger *Logger) InfoLogger() *log.Logger {
	return log.New(logger.InfoWriter(), "", 0)
}
func (logger *Logger) WarningLogger() *log.Logger {
	return log.New(logger.WarningWriter(), "", 0)
}
func (logger *Logger) ErrorLogger() *log.Logger {
	return log.New(logger.ErrorWriter(), "", 0)
}

// SINGLETON

func InitSingle(module wimark.Module) *Logger {
	if singleLogger == nil {
		singleLogger = Init(module)
	}
	return singleLogger
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

func Error(format string, values ...interface{}) {
	if singleLogger != nil {
		singleLogger.log(ErrorLevel, format, values...)
	}
}

func StopSingle() {
	if singleLogger != nil {
		singleLogger.Stop()
	}
	singleLogger = nil
}
