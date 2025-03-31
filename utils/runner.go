/*
  - Package utils
    @Author: zhizhuo
    @IDE：GoLand
    @File: runner.go
    @Date: 2025/3/10 下午2:11*
*/
package utils

import (
	"embed"
	"fmt"
	"gxx/types"
	"gxx/utils/cel"
	"gxx/utils/logger"
	"gxx/utils/pkg/finger"
	"gxx/utils/proto"
	"gxx/utils/request"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

//go:embed finger/*
var Fingers embed.FS
var AllFinger []*finger.Finger

// loadFingerprints 加载指纹规则文件
func loadFingerprints(options *types.CmdOptions) error {
	// 使用嵌入式指纹库
	if options.PocFile == "" && options.PocYaml == "" {
		entries, err := Fingers.ReadDir("finger")
		if err != nil {
			return fmt.Errorf("初始化finger目录出错: %v", err)
		}

		for _, entry := range entries {
			if isYamlFile(entry.Name()) {
				if poc, err := finger.Load(entry.Name(), Fingers); err == nil && poc != nil {
					AllFinger = append(AllFinger, poc)
				}
			}
		}
		return nil
	}

	var targetPath string

	if options.PocFile != "" {
		targetPath = options.PocFile
		logger.Info("加载yaml文件目录：", targetPath)

		// 遍历目录中的所有文件
		return filepath.Walk(targetPath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil {
				return err
			}
			if !info.IsDir() && isYamlFile(path) {
				if poc, err := finger.Read(path); err == nil && poc != nil {
					AllFinger = append(AllFinger, poc)
				}
			}
			return nil
		})

	} else if options.PocYaml != "" {
		targetPath = options.PocYaml
		logger.Info("加载yaml文件：", targetPath)

		// 直接读取单个文件
		if isYamlFile(targetPath) {
			if poc, err := finger.Read(targetPath); err == nil && poc != nil {
				AllFinger = append(AllFinger, poc)
			} else if err != nil {
				return fmt.Errorf("读取yaml文件出错: %v", err)
			}
		} else {
			return fmt.Errorf("%s 不是有效的yaml文件", targetPath)
		}
	}

	return nil
}

// isYamlFile 判断文件是否为YAML格式
func isYamlFile(filename string) bool {
	return strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml")
}

// prepareRequest 准备HTTP请求
func prepareRequest(target string) (*http.Request, error) {
	urlWithProtocol := target
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		urlWithProtocol = "https://" + target
	}

	req, err := http.NewRequest("GET", urlWithProtocol, nil)
	if err != nil {
		return nil, fmt.Errorf("创建临时请求失败: %v", err)
	}
	return req, nil
}

// evaluateFingerprint 评估单个指纹规则
func evaluateFingerprint(fg *finger.Finger, target, proxy string, customLib *cel.CustomLib) (bool, error) {
	SetiableMap := make(map[string]any)
	logger.Debug("获取指纹ID:", fg.Id)

	// 准备基础请求
	req, err := prepareRequest(target)
	if err != nil {
		return false, err
	}

	tempReqData, err := request.ParseRequest(req)
	if err != nil {
		return false, fmt.Errorf("解析请求失败: %v", err)
	}

	SetiableMap["request"] = tempReqData
	baseResponseMap := make(map[string]any)
	firstRequestSent := false
	ruleResults := make(map[string]bool)

	// 处理set规则
	if len(fg.Set) > 0 {
		finger.IsFuzzSet(fg.Set, SetiableMap, customLib)
	}

	// 评估规则
	for _, rule := range fg.Rules {
		needNewRequest := rule.Value.Request.Path != "/" && rule.Value.Request.Path != "" || !firstRequestSent

		if needNewRequest {
			logger.Debug("发送指纹探测请求")
			SetiableMaps, err := finger.SendRequest(target, rule.Value.Request, rule.Value, SetiableMap, proxy)
			if err != nil {
				logger.Debug(fmt.Sprintf("规则 %s 请求出错：%s", rule.Key, err.Error()))
				ruleResults[rule.Key] = false
				customLib.WriteRuleFunctionsROptions(rule.Key, false)
				continue
			}
			logger.Debug("请求发送完成，开始请求处理")
			if len(SetiableMaps) > 0 {
				SetiableMap = SetiableMaps
			}

			// 打印请求和响应的原始数据
			logger.Debug(fmt.Sprintf("请求RAW：\n%s", SetiableMaps["request"].(*proto.Request).Raw))
			logger.Debug(fmt.Sprintf("响应RAW：\n%s", SetiableMaps["response"].(*proto.Response).Raw))

			if !firstRequestSent {
				baseResponseMap = make(map[string]any)
				for k, v := range SetiableMap {
					baseResponseMap[k] = v
				}
				firstRequestSent = true
			}
		} else {
			for k, v := range baseResponseMap {
				SetiableMap[k] = v
			}
		}
		logger.Debug("请求完成，开始cel匹配处理")
		result, err := customLib.Evaluate(rule.Value.Expression, SetiableMap)
		if err != nil {
			logger.Debug(fmt.Sprintf("规则 %s CEL解析错误：%s", rule.Key, err.Error()))
			ruleResults[rule.Key] = false
		} else {
			ruleResults[rule.Key] = result.Value().(bool)
			logger.Debug(fmt.Sprintf("规则 %s 评估结果: %v", rule.Value.Expression, result.Value().(bool)))
		}

		customLib.WriteRuleFunctionsROptions(rule.Key, ruleResults[rule.Key])
	}

	// 最终评估
	result, err := customLib.Evaluate(fg.Expression, SetiableMap)
	if err != nil {
		return false, fmt.Errorf("最终表达式解析错误：%v", err)
	}

	finalResult := result.Value().(bool)
	logger.Debug(fmt.Sprintf("最终规则 %s 评估结果: %v", fg.Expression, finalResult))
	return finalResult, nil
}

// writeResult 写入结果到文件
func writeResult(output, format, target string, fg *finger.Finger, finalResult bool) error {
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

// NewFingerRunner 创建并运行指纹识别器
func NewFingerRunner(options *types.CmdOptions) {
	var target string
	if len(options.Target) == 0 && options.TargetsFile == "" {
		fmt.Println("错误: Target 和 TargetsFile 必填其中一项")
		return
	}

	if len(options.Target) > 0 {
		target = options.Target[0]
	} else if options.TargetsFile != "" {
		target = options.TargetsFile
	}

	proxy := options.Proxy
	logger.Debug(fmt.Sprintf("Target: %s, Proxy: %s", target, proxy))

	// 加载指纹规则
	if err := loadFingerprints(options); err != nil {
		logger.Error("加载指纹规则出错")
		return
	}

	// 创建工作池
	workerCount := options.Threads
	logger.Info(fmt.Sprintf("使用工作线程：%v个", workerCount))
	if options.Threads > 0 {
		workerCount = options.Threads
	}

	// 创建任务通道
	taskChan := make(chan *finger.Finger, len(AllFinger))
	// 创建结果通道
	resultChan := make(chan error, len(AllFinger))

	// 启动工作协程
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// 每个工作协程创建自己的CustomLib实例
			customLib := cel.NewCustomLib()

			for fg := range taskChan {
				result, err := evaluateFingerprint(fg, target, proxy, customLib)
				if err != nil {
					resultChan <- fmt.Errorf("指纹 %s 评估失败: %v", fg.Id, err)
					continue
				}

				// 如果指纹匹配成功，写入结果
				if result {
					fmt.Println(fmt.Sprintf("URL：%s  指纹：%s  匹配结果：%v", target, fg.Info.Name, result))
					if err := writeResult(options.Output, options.OutputFormat, target, fg, result); err != nil {
						resultChan <- fmt.Errorf("写入结果失败: %v", err)
					}
				}
			}
		}()
	}

	// 发送任务
	for _, fg := range AllFinger {
		taskChan <- fg
	}
	close(taskChan)

	// 等待所有工作协程完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

}
