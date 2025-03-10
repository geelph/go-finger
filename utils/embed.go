/*
  - Package finger
    @Author: zhizhuo
    @IDE：GoLand
    @File: embed.go
    @Date: 2025/3/10 下午4:37*
*/
package utils

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed finger/*
var EmbeddedFingerFS embed.FS

var hasEmbeddedFingers bool

func init() {
	files, err := EmbeddedFingerFS.ReadDir("finger")
	if err != nil || len(files) == 0 {
		hasEmbeddedFingers = false
		fmt.Printf("提示: 未嵌入指纹库，将使用文件系统中的指纹库。错误信息：%v\n", err)
		if len(files) == 0 {
			fmt.Println("提示：指纹目录为空。")
		}
	} else {
		hasEmbeddedFingers = true
	}
}

func GetFingerPath() string {
	if hasEmbeddedFingers {
		fmt.Println("使用嵌入的指纹库路径")
		return "embedded://finger/"
	}
	fmt.Println("使用文件系统中的指纹库路径")
	return "finger/"
}

func ExtractEmbeddedFingers() (string, error) {
	if !hasEmbeddedFingers {
		return "", fmt.Errorf("没有嵌入的指纹库")
	}

	tempDir, err := os.MkdirTemp("", "gxx-fingers-*")
	if err != nil {
		return "", fmt.Errorf("创建临时目录失败: %v", err)
	}

	err = fs.WalkDir(EmbeddedFingerFS, "finger", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		targetPath := filepath.Join(tempDir, path)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		data, err := EmbeddedFingerFS.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(targetPath, data, 0644)
	})

	if err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("提取指纹库失败: %v", err)
	}

	return tempDir, nil
}
