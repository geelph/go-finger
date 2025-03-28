package output

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	outputFile *os.File
	csvWriter  *csv.Writer
	mu         sync.Mutex
)

// InitOutput 初始化输出文件
func InitOutput(outputPath, format string) error {
	if outputPath == "" {
		return nil
	}

	// 确保输出目录存在
	dir := filepath.Dir(outputPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建输出目录失败: %v", err)
		}
	}

	// 打开文件（追加模式）
	file, err := os.OpenFile(outputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开输出文件失败: %v", err)
	}

	outputFile = file
	if format == "csv" {
		csvWriter = csv.NewWriter(file)
	}

	return nil
}

// Write 写入数据到文件
func Write(data []string, format string) error {
	if outputFile == nil {
		return nil
	}

	mu.Lock()
	defer mu.Unlock()

	if format == "csv" {
		if err := csvWriter.Write(data); err != nil {
			return fmt.Errorf("写入CSV数据失败: %v", err)
		}
		csvWriter.Flush()
	} else {
		// 默认txt格式
		line := fmt.Sprintf("%s\n", data[0])
		if _, err := outputFile.WriteString(line); err != nil {
			return fmt.Errorf("写入文本数据失败: %v", err)
		}
	}

	return nil
}

// Close 关闭输出文件
func Close() error {
	if outputFile != nil {
		if csvWriter != nil {
			csvWriter.Flush()
		}
		return outputFile.Close()
	}
	return nil
}
