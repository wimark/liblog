package liblog

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"
	"unicode/utf8"
)

type LogLevel int

var DebugLevel LogLevel = LogLevel(0)
var InfoLevel LogLevel = LogLevel(1)
var WarningLevel LogLevel = LogLevel(2)
var ErrorLevel LogLevel = LogLevel(3)

func (l LogLevel) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarningLevel:
		return "WARNING"
	case ErrorLevel:
		return "ERROR"
	}
	return fmt.Sprintf("LEVEL%d", l)
}

type LogMsg struct {
	Level    LogLevel
	format   string
	values   []interface{}
	Module   string
	ModuleId string
	SrcFile  string
	SrcLine  int
}

type Logger struct {
	module  string
	id      string
	output  chan *LogMsg
	Level   LogLevel
	writers []io.Writer
	stop    chan bool
	wg      sync.WaitGroup
	msgPool *sync.Pool
	bufPool *sync.Pool
}

var singleLogger *Logger

func (logger *Logger) worker() {
	defer logger.wg.Done()
	for msg := range logger.output {
		logger.writeMessage(msg)
		// Clear message values before putting back to pool to not hold references
		msg.values = nil
		logger.msgPool.Put(msg) // Return the message to the pool
	}
}

func (logger *Logger) writeMessage(msg *LogMsg) {
	if msg.Level < logger.Level {
		return
	}

	buf := logger.bufPool.Get().(*bytes.Buffer)
	buf.Reset()

	buf.WriteString(`{"timestamp":"`)
	buf.WriteString(time.Now().Format(time.RFC3339Nano))
	buf.WriteString(`","level":"`)
	buf.WriteString(msg.Level.String())
	buf.WriteString(`","message":"`)

	// Use a temporary buffer from the pool to format the message, avoiding allocation.
	tmpBuf := logger.bufPool.Get().(*bytes.Buffer)
	tmpBuf.Reset()
	fmt.Fprintf(tmpBuf, msg.format, msg.values...)

	// Escape the formatted message from the temporary buffer into the main buffer.
	messageBytes := tmpBuf.Bytes()
	for i := 0; i < len(messageBytes); {
		r, size := utf8.DecodeRune(messageBytes[i:])
		switch r {
		case '"', '\\':
			buf.WriteByte('\\')
			buf.WriteByte(byte(r))
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		default:
			buf.WriteRune(r)
		}
		i += size
	}
	logger.bufPool.Put(tmpBuf) // Return temporary buffer to the pool.

	buf.WriteString(`","service":"`)
	buf.WriteString(logger.module)
	if logger.id != "" {
		buf.WriteString(`","service_id":"`)
		buf.WriteString(logger.id)
	}
	if msg.SrcFile != "" {
		buf.WriteString(`","src_file":"`)
		buf.WriteString(msg.SrcFile)
		buf.WriteString(`","src_line":`)
		buf.WriteString(strconv.Itoa(msg.SrcLine))
	}
	buf.WriteString("}\n")

	// Write to all outputs
	os.Stdout.Write(buf.Bytes())
	for _, w := range logger.writers {
		w.Write(buf.Bytes())
	}
	logger.bufPool.Put(buf)
}

func (logger *Logger) log(level LogLevel, format string, values ...interface{}) {
	if level < logger.Level {
		return
	}

	msg := logger.msgPool.Get().(*LogMsg)
	msg.Level = level
	msg.format = format
	msg.values = values
	_, msg.SrcFile, msg.SrcLine, _ = runtime.Caller(2)
	msg.SrcFile = filepath.Base(msg.SrcFile)

	// Non-blocking send
	select {
	case logger.output <- msg:
	default:
		// Channel is full, drop the message and put it back to the pool
		logger.msgPool.Put(msg)
		log.Println("liblog: channel is full. Log message dropped.")
	}
}

// OBJECT

func Init(module string) *Logger {
	logger := &Logger{
		module:  module,
		output:  make(chan *LogMsg, 1024), // Use a buffered channel
		writers: make([]io.Writer, 0),
		stop:    make(chan bool),
		msgPool: &sync.Pool{
			New: func() interface{} {
				return &LogMsg{}
			},
		},
		bufPool: &sync.Pool{
			New: func() interface{} {
				// Pre-allocate buffer to a reasonable size to avoid re-allocations.
				b := new(bytes.Buffer)
				b.Grow(128)
				return b
			},
		},
	}
	level := os.Getenv("LOGLEVEL")
	switch level {
	case "ERROR", "3":
		logger.Level = ErrorLevel
	case "WARNING", "2":
		logger.Level = WarningLevel
	case "DEBUG", "0":
		logger.Level = DebugLevel
	default:
		logger.Level = InfoLevel
	}

	logger.wg.Add(1)
	go logger.worker()

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

func (logger *Logger) StopSync() {
	close(logger.output)
	logger.wg.Wait()
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
func (logger *Logger) DebugLogger(prefix string, flags int) *log.Logger {
	return log.New(logger.DebugWriter(), prefix, flags)
}
func (logger *Logger) InfoLogger(prefix string, flags int) *log.Logger {
	return log.New(logger.InfoWriter(), prefix, flags)
}
func (logger *Logger) WarningLogger(prefix string, flags int) *log.Logger {
	return log.New(logger.WarningWriter(), prefix, flags)
}
func (logger *Logger) ErrorLogger(prefix string, flags int) *log.Logger {
	return log.New(logger.ErrorWriter(), prefix, flags)
}

func (logger *Logger) AddWriter(writer io.Writer) {
	logger.writers = append(logger.writers, writer)
}

func (logger *Logger) SetModuleId(id string) {
	logger.id = id
}

// SINGLETON

func Singleton() *Logger {
	return singleLogger
}

func InitSingleStr(module string) *Logger {
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

func StopSyncSingle() {
	if singleLogger != nil {
		singleLogger.StopSync()
	}
	singleLogger = nil
}
