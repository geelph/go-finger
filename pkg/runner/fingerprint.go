package runner

import (
	"fmt"
	cel2 "gxx/pkg/cel"
	"gxx/pkg/finger"
	"gxx/pkg/network"
	"gxx/types"
	"gxx/utils"
	"gxx/utils/common"
	"gxx/utils/logger"
	"gxx/utils/proto"
	"os"
	"path/filepath"
)

// AllFinger 全局指纹数据
var AllFinger []*finger.Finger

// LoadFingerprints 加载指纹规则文件，支持从默认嵌入指纹库、指定目录或单个YAML文件加载
func LoadFingerprints(options types.YamlFingerType) error {
	// 使用嵌入式指纹库
	if options.PocFile == "" && options.PocYaml == "" {
		logger.Info("使用默认指纹库")
		fin, err := utils.GetFingerYaml()
		if err != nil {
			return err
		}
		AllFinger = fin
		return nil
	}

	// 从目录加载指纹文件
	if options.PocFile != "" {
		logger.Info(fmt.Sprintf("加载yaml文件目录：%s", options.PocFile))

		return filepath.WalkDir(options.PocFile, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && common.IsYamlFile(path) {
				if poc, err := finger.Read(path); err == nil && poc != nil {
					AllFinger = append(AllFinger, poc)
				}
			}
			return nil
		})
	}

	// 加载单个指纹文件
	if options.PocYaml != "" {
		logger.Info(fmt.Sprintf("加载yaml文件：%s", options.PocYaml))

		if !common.IsYamlFile(options.PocYaml) {
			return fmt.Errorf("%s 不是有效的yaml文件", options.PocYaml)
		}

		poc, err := finger.Read(options.PocYaml)
		if err != nil {
			return fmt.Errorf("读取yaml文件出错: %v", err)
		}

		if poc != nil {
			AllFinger = append(AllFinger, poc)
		}
	}

	return nil
}

// evaluateFingerprintWithCache 使用缓存的基础信息评估指纹规则，执行单个指纹的识别逻辑，包括发送请求和规则评估
func evaluateFingerprintWithCache(fg *finger.Finger, target string, baseInfo *BaseInfo, proxy string, timeout int, targetResult *TargetResult) (*FingerMatch, error) {
	customLib := cel2.NewCustomLib()
	// 初始化变量映射
	resultData := &FingerMatch{
		Finger: fg,
	}
	varMap := make(map[string]any)

	logger.Debug(fmt.Sprintf("获取指纹ID：%s", fg.Id))

	// 准备基础请求
	req, err := prepareRequest(target)
	if err != nil {
		return nil, err
	}

	tempReqData, err := network.ParseRequest(req)
	if err != nil {
		return nil, fmt.Errorf("解析请求失败: %v", err)
	}

	// 设置基础变量
	varMap["request"] = tempReqData
	varMap["title"] = baseInfo.Title
	varMap["server"] = baseInfo.Server

	// 初始化响应对象
	varMap["response"] = &proto.Response{
		Status:      baseInfo.StatusCode,
		Headers:     map[string]string{},
		ContentType: "",
		Body:        []byte{},
		Raw:         []byte{},
		RawHeader:   []byte{},
		Url:         &proto.UrlType{},
		Latency:     0,
	}

	// 处理预设规则
	if len(fg.Set) > 0 {
		finger.IsFuzzSet(fg.Set, varMap, customLib)
	}
	if len(fg.Payloads.Payloads) > 0 {
		finger.IsFuzzSet(fg.Payloads.Payloads, varMap, customLib)
	}

	// 评估规则
	for _, rule := range fg.Rules {
		// 优先使用缓存
		var lastRequest *proto.Request
		var lastResponse *proto.Response

		// 安全地获取缓存 - 避免频繁加锁解锁
		isCache, cache := ShouldUseCache(rule, targetResult)
		if isCache {
			lastRequest = cache.Request
			lastResponse = cache.Response

			if lastRequest != nil {
				varMap["request"] = lastRequest
			}
			if lastResponse != nil {
				varMap["response"] = lastResponse
			}
		}

		if lastRequest == nil || lastResponse == nil {
			// 发送新请求
			newVarMap, err := finger.SendRequest(target, rule.Value.Request, rule.Value, varMap, proxy, timeout)
			if err != nil {
				customLib.WriteRuleFunctionsROptions(rule.Key, false)
				continue
			}

			// 更新变量映射
			if len(newVarMap) > 0 {
				varMap = newVarMap
				if rule.Value.Request.FollowRedirects != false {
					if req, ok := varMap["request"].(*proto.Request); ok {
						targetResult.LastRequest = req
					}
					if resp, ok := varMap["response"].(*proto.Response); ok {
						targetResult.LastResponse = resp
					}
					// 使用线程安全的UpdateTargetCache函数
					UpdateTargetCache(varMap, targetResult)
				}
			}
		}

		// 调试信息输出
		logger.Debug(fmt.Sprintf("请求数据包：\n%s", varMap["request"].(*proto.Request).Raw))
		logger.Debug(fmt.Sprintf("响应数据包：\n%s", varMap["response"].(*proto.Response).Raw))
		logger.Debug("开始cel匹配处理")

		// 执行规则评估
		result, err := customLib.Evaluate(rule.Value.Expression, varMap)
		if err != nil {
			logger.Debug(fmt.Sprintf("规则 %s CEL解析错误：%s", rule.Key, err.Error()))
			customLib.WriteRuleFunctionsROptions(rule.Key, false)
		} else {
			ruleBool := result.Value().(bool)
			logger.Debug(fmt.Sprintf("规则 %s 评估结果: %v", rule.Value.Expression, ruleBool))
			customLib.WriteRuleFunctionsROptions(rule.Key, ruleBool)
		}

		// 处理输出规则
		if len(rule.Value.Output) > 0 {
			finger.IsFuzzSet(rule.Value.Output, varMap, customLib)
		}
	}

	// 执行最终评估
	result, err := customLib.Evaluate(fg.Expression, varMap)
	if err != nil {
		return nil, fmt.Errorf("最终表达式解析错误：%v", err)
	}

	customLib.Reset()
	resultData.Result = result.Value().(bool)
	resultData.Request = targetResult.LastRequest
	resultData.Response = targetResult.LastResponse

	logger.Debug(fmt.Sprintf("最终规则 %s 评估结果: %v", fg.Expression, resultData.Result))

	return resultData, nil
}
