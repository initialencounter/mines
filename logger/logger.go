package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Logger struct {
	mu         sync.Mutex
	dir        string
	file       *os.File
	currentDay string
}

var std *Logger

// Init 初始化日志模块，在可执行文件目录下创建 logs 文件夹
func Init() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}
	logDir := filepath.Join(filepath.Dir(exePath), "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}

	std = &Logger{dir: logDir}
	return nil
}

func (l *Logger) rotate() (*os.File, error) {
	today := time.Now().Format("2006-01-02")
	if l.currentDay == today && l.file != nil {
		return l.file, nil
	}
	if l.file != nil {
		l.file.Close()
	}
	f, err := os.OpenFile(filepath.Join(l.dir, today+".log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	l.file = f
	l.currentDay = today
	return f, nil
}

func (l *Logger) write(level, ip, uid, user, action, detail string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	f, err := l.rotate()
	if err != nil {
		return
	}

	ts := time.Now().Format("2006-01-02 15:04:05")
	line := fmt.Sprintf("[%s] [%s] [IP:%s] [UID:%s] [User:%s] %s - %s\n",
		ts, level, ip, uid, user, action, detail)
	f.WriteString(line)
}

// Info 记录 INFO 级别日志
func Info(ip, uid, user, action, detail string) {
	if std != nil {
		std.write("INFO", ip, uid, user, action, detail)
	}
}

// Warn 记录 WARN 级别日志
func Warn(ip, uid, user, action, detail string) {
	if std != nil {
		std.write("WARN", ip, uid, user, action, detail)
	}
}
