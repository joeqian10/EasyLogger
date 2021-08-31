package EasyLogger

import (
	"bytes"
	"fmt"
	"github.com/gookit/color"
	"github.com/natefinch/lumberjack"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const (
	CALL_DEPTH = 2

	TRACE = "[TRACE]"
	DEBUG = "[DEBUG]"
	INFO  = "[INFO ]"
	WARN  = "[WARN ]"
	ERROR = "[ERROR]"
	FATAL = "[FATAL]"
)

var (
	Trace = color.Cyan
	Debug = color.Blue
	Info  = color.Green
	Warn  = color.Yellow
	Error = color.Red
	Fatal = color.Magenta
)

func GetGID() uint64 {
	var buf [64]byte
	b := buf[:runtime.Stack(buf[:], false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

// EasyLogger uses log.Logger inside
type EasyLogger struct {
	logger *log.Logger
}

func NewRotatingEasyLogger(fileName string,
	maxFileSize int,
	maxBackupAge int,
	maxBackupFiles int,
	useLocalTime bool,
	useCompression bool,
	lineFlag int, // log.Ldate|log.Lmicroseconds
	prefixForLogger string,
	needConsoleOut bool) *EasyLogger {

	lum := &lumberjack.Logger{
		Filename:   fileName,
		MaxSize:    maxFileSize,    // defaults to 100 megabytes
		MaxAge:     maxBackupAge,   // maximum number of days to retain old log files based on the timestamp encoded in their filename, default is not to remove old log files
		MaxBackups: maxBackupFiles, // maximum number of old log files to retain, default is to retain all old log files
		LocalTime:  useLocalTime,   // default is to use UTC time
		Compress:   useCompression, // compress the rotated files, default is not to compress
	}
	ws := []io.Writer{lum}
	if needConsoleOut {
		ws = append(ws, os.Stdout)
	}
	outs := io.MultiWriter(ws...)
	ll := log.New(outs, prefixForLogger, lineFlag)

	return &EasyLogger{logger: ll}
}

func (this *EasyLogger) output(level string, a ...interface{}) error {
	gid := GetGID()
	gidStr := strconv.FormatUint(gid, 10)

	a = append([]interface{}{level, "GID", gidStr + ","}, a...)

	return this.logger.Output(CALL_DEPTH, fmt.Sprintln(a...))
}

func (this *EasyLogger) outputf(level string, format string, v ...interface{}) error {
	gid := GetGID()
	v = append([]interface{}{level, "GID", gid}, v...)

	return this.logger.Output(CALL_DEPTH, fmt.Sprintf("%s %s %d, "+format+"\n", v...))
}

func (this *EasyLogger) Trace(a ...interface{}) {
	pc := make([]uintptr, 10)
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	file, line := f.FileLine(pc[0])
	fileName := filepath.Base(file)

	nameFull := f.Name()
	nameEnd := filepath.Ext(nameFull)
	funcName := strings.TrimPrefix(nameEnd, ".")

	a = append([]interface{}{funcName + "()", fileName + ":" + strconv.Itoa(line)}, a...)
	this.output(Trace.Sprint(TRACE), a...)
}

func (this *EasyLogger) Tracef(format string, a ...interface{}) {
	pc := make([]uintptr, 10)
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	file, line := f.FileLine(pc[0])
	fileName := filepath.Base(file)

	nameFull := f.Name()
	nameEnd := filepath.Ext(nameFull)
	funcName := strings.TrimPrefix(nameEnd, ".")

	a = append([]interface{}{funcName, fileName, line}, a...)
	this.outputf(Trace.Sprint(TRACE), "%s() %s:%d "+format, a...)
}

func (this *EasyLogger) Debug(a ...interface{}) {
	pc := make([]uintptr, 10)
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	file, line := f.FileLine(pc[0])
	fileName := filepath.Base(file)

	a = append([]interface{}{f.Name(), fileName + ":" + strconv.Itoa(line)}, a...)
	this.output(Debug.Sprint(DEBUG), a...)
}

func (this *EasyLogger) Debugf(format string, a ...interface{}) {
	pc := make([]uintptr, 10)
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	file, line := f.FileLine(pc[0])
	fileName := filepath.Base(file)

	a = append([]interface{}{f.Name(), fileName, line}, a...)
	this.outputf(Debug.Sprint(DEBUG), "%s() %s:%d "+format, a...)
}

func (this *EasyLogger) Info(a ...interface{}) {
	this.output(Info.Sprint(INFO), a...)
}

func (this *EasyLogger) Infof(format string, a ...interface{}) {
	this.outputf(Info.Sprint(INFO), format, a...)
}

func (this *EasyLogger) Warn(a ...interface{}) {
	this.output(Warn.Sprint(WARN), a...)
}

func (this *EasyLogger) Warnf(format string, a ...interface{}) {
	this.outputf(Warn.Sprint(WARN), format, a...)
}

func (this *EasyLogger) Error(a ...interface{}) {
	this.output(Error.Sprint(ERROR), a...)
}

func (this *EasyLogger) Errorf(format string, a ...interface{}) {
	this.outputf(Error.Sprint(ERROR), format, a...)
}

func (this *EasyLogger) Fatal(a ...interface{}) {
	this.output(Fatal.Sprint(FATAL), a...)
}

func (this *EasyLogger) Fatalf(format string, a ...interface{}) {
	this.outputf(Fatal.Sprint(FATAL), format, a...)
}
