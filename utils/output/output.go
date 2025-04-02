package output

import (
	"encoding/csv"
	"fmt"
	finger2 "gxx/pkg/finger"
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

// WriteResult 写入结果到文件
func WriteResult(output, format, target string, fg *finger2.Finger, finalResult bool) error {
	if output == "" {
		return nil
	}
	// 确保输出目录存在
	dir := filepath.Dir(output)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建输出目录失败: %v", err)
		}
	}

	// 打开文件（追加模式）
	file, err := os.OpenFile(output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开输出文件失败: %v", err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	// 格式化结果
	var line string
	if format == "csv" {
		line = fmt.Sprintf("%s,%s,%s,%v\n", target, fg.Id, fg.Info.Name, finalResult)
	} else {
		line = fmt.Sprintf("URL: %s\t指纹ID: %s\t指纹名称: %s\t匹配结果: %v\n", target, fg.Id, fg.Info.Name, finalResult)
	}

	// 写入结果
	if _, err := file.WriteString(line); err != nil {
		return fmt.Errorf("写入结果失败: %v", err)
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
