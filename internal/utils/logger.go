package utils

import (
	"fmt"
	"io"
	"log"
	"os"
)

type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

type Logger struct {
	level      LogLevel
	debugLog   *log.Logger
	infoLog    *log.Logger
	warnLog    *log.Logger
	errorLog   *log.Logger
}

var defaultLogger *Logger

func init() {
	defaultLogger = NewLogger(InfoLevel, os.Stdout)
}

func NewLogger(level LogLevel, output io.Writer) *Logger {
	return &Logger{
		level:    level,
		debugLog: log.New(output, "[DEBUG] ", log.Ldate|log.Ltime|log.Lshortfile),
		infoLog:  log.New(output, "[INFO] ", log.Ldate|log.Ltime),
		warnLog:  log.New(output, "[WARN] ", log.Ldate|log.Ltime),
		errorLog: log.New(output, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

func SetLogLevel(level LogLevel) {
	defaultLogger.level = level
}

func SetLogLevelFromString(levelStr string) {
	switch levelStr {
	case "debug":
		SetLogLevel(DebugLevel)
	case "info":
		SetLogLevel(InfoLevel)
	case "warn":
		SetLogLevel(WarnLevel)
	case "error":
		SetLogLevel(ErrorLevel)
	default:
		SetLogLevel(InfoLevel)
	}
}

func (l *Logger) Debug(v ...interface{}) {
	if l.level <= DebugLevel {
		l.debugLog.Output(2, fmt.Sprintln(v...))
	}
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	if l.level <= DebugLevel {
		l.debugLog.Output(2, fmt.Sprintf(format, v...))
	}
}

func (l *Logger) Info(v ...interface{}) {
	if l.level <= InfoLevel {
		l.infoLog.Output(2, fmt.Sprintln(v...))
	}
}

func (l *Logger) Infof(format string, v ...interface{}) {
	if l.level <= InfoLevel {
		l.infoLog.Output(2, fmt.Sprintf(format, v...))
	}
}

func (l *Logger) Warn(v ...interface{}) {
	if l.level <= WarnLevel {
		l.warnLog.Output(2, fmt.Sprintln(v...))
	}
}

func (l *Logger) Warnf(format string, v ...interface{}) {
	if l.level <= WarnLevel {
		l.warnLog.Output(2, fmt.Sprintf(format, v...))
	}
}

func (l *Logger) Error(v ...interface{}) {
	if l.level <= ErrorLevel {
		l.errorLog.Output(2, fmt.Sprintln(v...))
	}
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	if l.level <= ErrorLevel {
		l.errorLog.Output(2, fmt.Sprintf(format, v...))
	}
}

// Package-level functions that use the default logger
func Debug(v ...interface{}) {
	defaultLogger.Debug(v...)
}

func Debugf(format string, v ...interface{}) {
	defaultLogger.Debugf(format, v...)
}

func Info(v ...interface{}) {
	defaultLogger.Info(v...)
}

func Infof(format string, v ...interface{}) {
	defaultLogger.Infof(format, v...)
}

func Warn(v ...interface{}) {
	defaultLogger.Warn(v...)
}

func Warnf(format string, v ...interface{}) {
	defaultLogger.Warnf(format, v...)
}

func Error(v ...interface{}) {
	defaultLogger.Error(v...)
}

func Errorf(format string, v ...interface{}) {
	defaultLogger.Errorf(format, v...)
}