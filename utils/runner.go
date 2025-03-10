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
)

//go:embed finger/*
var Fingers embed.FS
var AllFinger []*finger.Finger

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
	// 判断是否有协议，没有默认使用https
	var urlWithProtocol = target
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		urlWithProtocol = "https://" + target
	}
	customLib := cel.NewCustomLib()

	// 检查 PocFile 和 PocYaml 是否都为空
	if options.PocFile == "" {
		entries, err := Fingers.ReadDir("finger") // 修改为正确的目录名
		if err != nil {
			fmt.Println("初始化finger目录出错", err)
			return
		}
		for _, one := range entries {
			path := one.Name()
			if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
				if poc, _ := finger.Load(path, Fingers); poc != nil {
					AllFinger = append(AllFinger, poc)
				}
			}
		}
	} else {
		fmt.Println("加载yaml文件，:", options.PocFile)
		err := filepath.Walk("finger",
			func(path string, info os.FileInfo, err error) error {
				if err != nil || info == nil {
					return err
				}
				if !info.IsDir() {
					if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
						poc, _ := finger.Read(path)
						if poc != nil {
							AllFinger = append(AllFinger, poc)
						}
					}
				}
				return nil
			})
		if err != nil {
			fmt.Println("加载yaml文件失败", err)
		}
	}

	for _, fg := range AllFinger {
		var SetiableMap = make(map[string]any)
		fmt.Println("获取指纹ID:", fg.Id)
		// 生成临时的request请求
		tempReq, err := http.NewRequest("GET", urlWithProtocol, nil)
		if err != nil {
			fmt.Println("创建临时请求失败:", err)
			return
		}
		tempReqData, err := request.ParseRequest(tempReq)
		if err != nil {
			fmt.Println("解析请求失败:", err)
			return
		}
		fmt.Println(tempReqData)
		SetiableMap["request"] = tempReqData

		// 处理yaml中set
		if len(fg.Set) > 0 {
			finger.IsFuzzSet(fg.Set, SetiableMap, customLib)
		}

		// 发送第一个请求，获取基础响应数据
		var baseResponseMap map[string]any
		var firstRequestSent bool = false

		// 创建规则结果映射
		ruleResults := make(map[string]bool)

		for _, rule := range fg.Rules {
			// 检查是否需要发送新请求
			needNewRequest := true

			// 如果rule.Value.Request.Path是/或为空，且已经发送过请求，则复用之前的请求数据
			if (rule.Value.Request.Path == "/" || rule.Value.Request.Path == "") && firstRequestSent {
				needNewRequest = false
				fmt.Printf("规则 %s 复用之前的请求数据\n", rule.Key)
				// 复用之前的请求数据
				for k, v := range baseResponseMap {
					SetiableMap[k] = v
				}
			}

			// 如果需要发送新请求
			if needNewRequest {
				fmt.Printf("规则 %s 发送新请求: %s\n", rule.Key, rule.Value.Request.Path)
				var err error
				SetiableMap, err = finger.SendRequest(target, rule.Value.Request, rule.Value, SetiableMap, proxy)
				if err != nil {
					fmt.Printf("规则 %s 请求出错：%s\n", rule.Key, err.Error())
					ruleResults[rule.Key] = false
					customLib.WriteRuleFunctionsROptions(rule.Key, false)
					continue
				}

				// 如果是第一次发送请求，保存响应数据
				if !firstRequestSent {
					baseResponseMap = make(map[string]any)
					for k, v := range SetiableMap {
						baseResponseMap[k] = v
					}
					firstRequestSent = true
				}
			}

			// 评估表达式
			result, err := customLib.Evaluate(rule.Value.Expression, SetiableMap)
			if err != nil {
				fmt.Printf("规则 %s CEL解析错误：%s\n", rule.Key, err.Error())
				ruleResults[rule.Key] = false
			} else {
				// 保存规则结果
				ruleResults[rule.Key] = result.Value().(bool)
				// 打印评估结果
				fmt.Printf("规则 %s 评估结果: %v\n", rule.Value.Expression, result.Value().(bool))
			}

			// 写入规则结果
			customLib.WriteRuleFunctionsROptions(rule.Key, ruleResults[rule.Key])
		}

		// 最终评估
		result, err := customLib.Evaluate(fg.Expression, SetiableMap)
		if err != nil {
			fmt.Println("最终表达式解析错误：", err.Error())
			return
		}

		// 打印评估结果
		fmt.Printf("最终规则 %s 评估结果: %v\n", fg.Expression, result.Value().(bool))

		// 打印所有规则结果摘要
		fmt.Println("\n规则执行结果摘要:")
		for ruleName, ruleResult := range ruleResults {
			fmt.Printf("  %s: %v\n", ruleName, ruleResult)
		}
	}
}
