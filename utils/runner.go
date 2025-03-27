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
	"gxx/utils/pkg/finger"
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
	if options.PocFile == "" {
		entries, err := Fingers.ReadDir("finger")
		if err != nil {
			return fmt.Errorf("初始化finger目录出错: %v", err)
		}

		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".yaml") || strings.HasSuffix(entry.Name(), ".yml") {
				if poc, err := finger.Load(entry.Name(), Fingers); err == nil && poc != nil {
					AllFinger = append(AllFinger, poc)
				}
			}
		}
		return nil
	}

	fmt.Println("加载yaml文件:", options.PocFile)
	return filepath.Walk("finger", func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return err
		}
		if !info.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			if poc, err := finger.Read(path); err == nil && poc != nil {
				AllFinger = append(AllFinger, poc)
			}
		}
		return nil
	})
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
func evaluateFingerprint(fg *finger.Finger, target, proxy string, customLib *cel.CustomLib) error {
	SetiableMap := make(map[string]any)
	fmt.Println("获取指纹ID:", fg.Id)

	// 准备基础请求
	req, err := prepareRequest(target)
	if err != nil {
		return err
	}

	tempReqData, err := request.ParseRequest(req)
	if err != nil {
		return fmt.Errorf("解析请求失败: %v", err)
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
			SetiableMaps, err := finger.SendRequest(target, rule.Value.Request, rule.Value, SetiableMap, proxy)
			if err != nil {
				fmt.Printf("规则 %s 请求出错：%s\n", rule.Key, err.Error())
				ruleResults[rule.Key] = false
				customLib.WriteRuleFunctionsROptions(rule.Key, false)
				continue
			}
			if len(SetiableMaps) > 0 {
				SetiableMap = SetiableMaps
			}
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

		result, err := customLib.Evaluate(rule.Value.Expression, SetiableMap)
		if err != nil {
			fmt.Printf("规则 %s CEL解析错误：%s\n", rule.Key, err.Error())
			ruleResults[rule.Key] = false
		} else {
			ruleResults[rule.Key] = result.Value().(bool)
			fmt.Printf("规则 %s 评估结果: %v\n", rule.Value.Expression, result.Value().(bool))
		}

		customLib.WriteRuleFunctionsROptions(rule.Key, ruleResults[rule.Key])
	}

	// 最终评估
	result, err := customLib.Evaluate(fg.Expression, SetiableMap)
	if err != nil {
		return fmt.Errorf("最终表达式解析错误：%v", err)
	}

	fmt.Printf("最终规则 %s 评估结果: %v\n", fg.Expression, result.Value().(bool))
	fmt.Println("\n规则执行结果摘要:")
	for ruleName, ruleResult := range ruleResults {
		fmt.Printf("  %s: %v\n", ruleName, ruleResult)
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
	fmt.Println("target:", target, "proxy:", proxy)

	// 加载指纹规则
	if err := loadFingerprints(options); err != nil {
		fmt.Println(err)
		return
	}

	customLib := cel.NewCustomLib()
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5) // 限制并发数

	for _, fg := range AllFinger {
		wg.Add(1)
		semaphore <- struct{}{} // 获取信号量

		go func(f *finger.Finger) {
			defer wg.Done()
			defer func() { <-semaphore }() // 释放信号量

			if err := evaluateFingerprint(f, target, proxy, customLib); err != nil {
				fmt.Printf("指纹 %s 评估失败: %v\n", f.Id, err)
			}
		}(fg)
	}

	wg.Wait()
}
