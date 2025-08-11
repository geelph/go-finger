package logger

import (
	"fmt"
	"io"
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
	originalOutput io.Writer  // 保存原始的终端输出
	outputMutex    sync.Mutex // 保护输出切换的互斥锁
}

var (
	once     sync.Once
	instance *Logger
	// 默认日志级别
	defaultLogger = &Logger{
		terminalLogger: logrus.New(),
		fileLogger:     logrus.New(),
		logLevel:       logrus.InfoLevel,
		originalOutput: os.Stdout,
	}
	// 空输出对象，用于暂停终端日志
	nullOutput = io.Discard
)

// InitLogger 初始化全局日志实例
func InitLogger(logDir string, maxFiles int, logLevel int, noFileLog ...bool) {
	once.Do(func() {
		instance = NewLogger(logDir, maxFiles, logLevel, noFileLog...)
	})
}

func NewLogger(logDir string, maxFiles int, logLevel int, noFileLog ...bool) *Logger {
	terminalLogger := logrus.New()
	terminalFormatter := &CustomFormatter{IsColored: true}
	terminalLogger.SetFormatter(terminalFormatter)
	terminalLogger.SetOutput(os.Stdout)

	fileLogger := logrus.New()
	plainFormatter := &PlainFormatter{CustomFormatter: *terminalFormatter}
	fileLogger.SetFormatter(plainFormatter)

	// 默认启用文件日志
	disableFileLog := false
	if len(noFileLog) > 0 && noFileLog[0] {
		disableFileLog = true
	}

	if !disableFileLog {
		// 确保日志目录存在
		if _, err := os.Stat(logDir); os.IsNotExist(err) {
			if err := os.MkdirAll(logDir, 0755); err != nil {
				terminalLogger.Errorf("创建日志目录失败: %v", err)
			}
		}

		logFile := &lumberjack.Logger{
			Filename:   logDir + "/" + time.Now().Format("2006-01-02") + ".log",
			MaxBackups: maxFiles,
			MaxSize:    50, //日志最大大小为50MB
			MaxAge:     10, //保存10天
			Compress:   true,
		}
		fileLogger.SetOutput(logFile)
	} else {
		// 如果禁用文件日志，将文件日志输出设置为空
		fileLogger.SetOutput(io.Discard)
	}

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

	return &Logger{
		terminalLogger: terminalLogger,
		fileLogger:     fileLogger,
		logLevel:       level,
		originalOutput: os.Stdout,
	}
}

// PauseTerminalLogging 暂停终端日志输出，在显示进度条前调用
func PauseTerminalLogging() {
	if instance != nil {
		instance.PauseTerminalLogging()
	}
}

// ResumeTerminalLogging 恢复终端日志输出，在进度条完成后调用
func ResumeTerminalLogging() {
	if instance != nil {
		instance.ResumeTerminalLogging()
	}
}

// PauseTerminalLogging 暂停终端日志输出的实例方法
func (l *Logger) PauseTerminalLogging() {
	l.outputMutex.Lock()
	defer l.outputMutex.Unlock()
	l.terminalLogger.SetOutput(nullOutput)
}

// ResumeTerminalLogging 恢复终端日志输出的实例方法
func (l *Logger) ResumeTerminalLogging() {
	l.outputMutex.Lock()
	defer l.outputMutex.Unlock()
	l.terminalLogger.SetOutput(l.originalOutput)
}

func Info(format string, args ...interface{}) {
	if instance != nil {
		instance.Info(format, args...)
	} else {
		defaultLogger.Info(format, args...)
	}
}

func Error(format string, args ...interface{}) {
	if instance != nil {
		instance.Error(format, args...)
	} else {
		defaultLogger.Error(format, args...)
	}
}

func Warn(format string, args ...interface{}) {
	if instance != nil {
		instance.Warn(format, args...)
	} else {
		defaultLogger.Warn(format, args...)
	}
}

func Debug(format string, args ...interface{}) {
	if instance != nil {
		instance.Debug(format, args...)
	} else {
		defaultLogger.Debug(format, args...)
	}
}

func Success(format string, args ...interface{}) {
	if instance != nil {
		instance.Success(format, args...)
	} else {
		defaultLogger.Success(format, args...)
	}
}

func (l *Logger) Info(format string, args ...interface{}) {
	if l.logLevel >= logrus.InfoLevel {
		message := fmt.Sprintf(format, args...)
		l.terminalLogger.Info(message)
		l.fileLogger.Info(message)
	}
}

func (l *Logger) Error(format string, args ...interface{}) {
	if l.logLevel >= logrus.ErrorLevel {
		message := fmt.Sprintf(format, args...)
		l.terminalLogger.Error(message)
		l.fileLogger.Error(message)
	}
}

func (l *Logger) Warn(format string, args ...interface{}) {
	if l.logLevel >= logrus.WarnLevel {
		message := fmt.Sprintf(format, args...)
		l.terminalLogger.Warn(message)
		l.fileLogger.Warn(message)
	}
}

func (l *Logger) Debug(format string, args ...interface{}) {
	// 确保Debug级别的日志能够正确输出
	// 注意：logrus.DebugLevel的值比logrus.InfoLevel小
	if l.logLevel <= logrus.DebugLevel {
		message := fmt.Sprintf(format, args...)
		l.terminalLogger.Debug(message)
		l.fileLogger.Debug(message)
	}
}

func (l *Logger) Success(format string, args ...interface{}) {
	if l.logLevel >= logrus.InfoLevel {
		message := fmt.Sprintf(format, args...)
		l.terminalLogger.WithField("type", "success").Info(message)
		l.fileLogger.WithField("type", "success").Info(message)
	}
}
