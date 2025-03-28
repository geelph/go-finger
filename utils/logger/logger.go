/*
  - Package logger
    @Author: zhizhuo
    @IDE：GoLand
    @File: logger.go
    @Date: 2025/2/20 下午3:39*
*/
package logger

import (
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/mgutz/ansi"
	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
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
	logLevel       logrus.Level
}

var (
	once     sync.Once
	instance *Logger
	// 默认日志级别
	defaultLogger = &Logger{
		terminalLogger: logrus.New(),
		fileLogger:     logrus.New(),
		logLevel:       logrus.InfoLevel,
	}
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

	// 确保日志目录存在
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		if err := os.MkdirAll(logDir, 0755); err != nil {
			terminalLogger.Errorf("创建日志目录失败: %v", err)
		}
	}

	logFile := &lumberjack.Logger{
		Filename:   logDir + "/" + time.Now().Format("2006-01-02") + ".log",
		MaxBackups: maxFiles,
		MaxSize:    10,
		MaxAge:     30,
		Compress:   true,
	}
	fileLogger.SetOutput(logFile)

	// 设置日志级别
	var level logrus.Level
	switch logLevel {
	case 1:
		level = logrus.InfoLevel
	case 2:
		level = logrus.ErrorLevel
	case 3:
		level = logrus.WarnLevel
	case 4:
		level = logrus.DebugLevel
	case 5:
		level = logrus.TraceLevel
	default:
		level = logrus.InfoLevel
	}

	// 确保日志级别正确设置
	terminalLogger.SetLevel(level)
	fileLogger.SetLevel(level)

	// 验证日志级别是否正确设置
	if logLevel == 4 || logLevel == 5 {
		terminalLogger.Info("日志级别设置为:", level.String())
	}

	return &Logger{
		terminalLogger: terminalLogger,
		fileLogger:     fileLogger,
		logLevel:       level,
	}
}

func Info(args ...interface{}) {
	if instance != nil {
		instance.Info(args...)
	} else {
		defaultLogger.Info(args...)
	}
}

func Error(args ...interface{}) {
	if instance != nil {
		instance.Error(args...)
	} else {
		defaultLogger.Error(args...)
	}
}

func Warn(args ...interface{}) {
	if instance != nil {
		instance.Warn(args...)
	} else {
		defaultLogger.Warn(args...)
	}
}

func Debug(args ...interface{}) {
	if instance != nil {
		instance.Debug(args...)
	} else {
		defaultLogger.Debug(args...)
	}
}

func Success(args ...interface{}) {
	if instance != nil {
		instance.Success(args...)
	} else {
		defaultLogger.Success(args...)
	}
}

func (l *Logger) Info(args ...interface{}) {
	if l.logLevel >= logrus.InfoLevel {
		l.terminalLogger.Info(args...)
		l.fileLogger.Info(args...)
	}
}

func (l *Logger) Error(args ...interface{}) {
	if l.logLevel >= logrus.ErrorLevel {
		l.terminalLogger.Error(args...)
		l.fileLogger.Error(args...)
	}
}

func (l *Logger) Warn(args ...interface{}) {
	if l.logLevel >= logrus.WarnLevel {
		l.terminalLogger.Warn(args...)
		l.fileLogger.Warn(args...)
	}
}

func (l *Logger) Debug(args ...interface{}) {
	// 确保Debug级别的日志能够正确输出
	// 注意：logrus.DebugLevel的值比logrus.InfoLevel小
	if l.logLevel <= logrus.DebugLevel {
		l.terminalLogger.Debug(args...)
		l.fileLogger.Debug(args...)
	}
}

func (l *Logger) Success(args ...interface{}) {
	if l.logLevel >= logrus.InfoLevel {
		l.terminalLogger.WithField("type", "success").Info(args...)
		l.fileLogger.WithField("type", "success").Info(args...)
	}
}
