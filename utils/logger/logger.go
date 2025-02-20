/*
  - Package logger
    @Author: zhizhuo
    @IDE：GoLand
    @File: logger.go
    @Date: 2025/2/20 下午3:39*
*/
package logger

import (
	"github.com/mgutz/ansi"
	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

// CustomFormatter 自定义的日志格式化器
type CustomFormatter struct {
	IsColored bool
}

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logLevel := strings.ToUpper(entry.Level.String())
	message := entry.Message

	if entry.Data["type"] == "success" {
		logLevel = "SUCCESS"
	}

	if f.IsColored {
		var colorFunc func(string) string
		switch entry.Level {
		case logrus.ErrorLevel:
			colorFunc = ansi.ColorFunc("red")
		case logrus.WarnLevel:
			colorFunc = ansi.ColorFunc("yellow")
		case logrus.DebugLevel:
			colorFunc = ansi.ColorFunc("blue")
		case logrus.InfoLevel:
			colorFunc = func(s string) string { return s }
		default:
			colorFunc = func(s string) string { return s }
		}

		if entry.Data["type"] == "success" {
			colorFunc = ansi.ColorFunc("green")
		}

		logLevel = colorFunc(logLevel)
	}

	logMessage := "[" + timestamp + "] [" + logLevel + "] " + message + "\n"
	return []byte(logMessage), nil
}

type PlainFormatter struct {
	CustomFormatter
}

func (f *PlainFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	coloredMessage, err := f.CustomFormatter.Format(entry)
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	plainMessage := re.ReplaceAll(coloredMessage, []byte(""))

	return plainMessage, nil
}

type Logger struct {
	terminalLogger *logrus.Logger
	fileLogger     *logrus.Logger
}

var (
	once     sync.Once
	instance *Logger
)

func InitLogger(logDir string, maxFiles int, logLevel int) {
	once.Do(func() {
		instance = NewLogger(logDir, maxFiles, logLevel)
	})
}

func NewLogger(logDir string, maxFiles int, logLevel int) *Logger {
	terminalLogger := logrus.New()
	terminalFormatter := &CustomFormatter{IsColored: true}
	terminalLogger.SetFormatter(terminalFormatter)
	terminalLogger.SetOutput(os.Stdout)

	fileLogger := logrus.New()
	plainFormatter := &PlainFormatter{CustomFormatter: *terminalFormatter}
	fileLogger.SetFormatter(plainFormatter)

	logFile := &lumberjack.Logger{
		Filename:   logDir + "/" + time.Now().Format("2006-01-02") + ".log",
		MaxBackups: maxFiles,
		MaxSize:    10,
		MaxAge:     30,
		Compress:   true,
	}
	fileLogger.SetOutput(logFile)

	switch logLevel {
	case 1:
		terminalLogger.SetLevel(logrus.InfoLevel)
		fileLogger.SetLevel(logrus.InfoLevel)
	case 2:
		terminalLogger.SetLevel(logrus.ErrorLevel)
		fileLogger.SetLevel(logrus.ErrorLevel)
	case 3:
		terminalLogger.SetLevel(logrus.WarnLevel)
		fileLogger.SetLevel(logrus.WarnLevel)
	case 4:
		terminalLogger.SetLevel(logrus.DebugLevel)
		fileLogger.SetLevel(logrus.DebugLevel)
	case 5:
		terminalLogger.SetLevel(logrus.TraceLevel)
		fileLogger.SetLevel(logrus.TraceLevel)
	default:
		terminalLogger.SetLevel(logrus.InfoLevel)
		fileLogger.SetLevel(logrus.InfoLevel)
	}

	return &Logger{
		terminalLogger: terminalLogger,
		fileLogger:     fileLogger,
	}
}

func Info(args ...interface{}) {
	if instance != nil {
		instance.Info(args...)
	}
}

func Error(args ...interface{}) {
	if instance != nil {
		instance.Error(args...)
	}
}

func Warn(args ...interface{}) {
	if instance != nil {
		instance.Warn(args...)
	}
}

func Debug(args ...interface{}) {
	if instance != nil {
		instance.Debug(args...)
	}
}

func Success(args ...interface{}) {
	if instance != nil {
		instance.Success(args...)
	}
}

func (l *Logger) Info(args ...interface{}) {
	l.terminalLogger.Info(args...)
	l.fileLogger.Info(args...)
}

func (l *Logger) Error(args ...interface{}) {
	l.terminalLogger.Error(args...)
	l.fileLogger.Error(args...)
}

func (l *Logger) Warn(args ...interface{}) {
	l.terminalLogger.Warn(args...)
	l.fileLogger.Warn(args...)
}

func (l *Logger) Debug(args ...interface{}) {
	l.terminalLogger.Debug(args...)
	l.fileLogger.Debug(args...)
}

func (l *Logger) Success(args ...interface{}) {
	l.terminalLogger.WithField("type", "success").Info(args...)
	l.fileLogger.WithField("type", "success").Info(args...)
}
