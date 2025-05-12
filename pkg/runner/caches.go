/*
  - Package caches
    @Author: zhizhuo
    @IDE：GoLand
    @File: caches.go
    @Date: 2025/5/6 上午11:39*
*/
package runner

import (
	"fmt"
	"gxx/pkg/finger"
	"gxx/utils/common"
	"gxx/utils/logger"
	"gxx/utils/proto"
	"strconv"
	"strings"
	"sync"
)

// CacheRequest 存储请求和响应的缓存条目
type CacheRequest struct {
	Request  *proto.Request
	Response *proto.Response
}

// 全局缓存映射和保护它的互斥锁
var (
	cacheMap    = make(map[string]*CacheRequest, 1024) // 预分配更大空间以减少哈希表扩容
	cacheMapMux sync.RWMutex                           // 读写锁保护缓存映射
)

// GenerateCacheKey 生成缓存键
func GenerateCacheKey(target string, method string, followRedirects bool) string {
	return common.MD5Hash(target + ":" + method + ":" + strconv.FormatBool(followRedirects))
}

// ShouldUseCache 判断是否应该使用缓存，对于根路径的GET请求，可以重用缓存的请求和响应
func ShouldUseCache(rule finger.RuleMap, target string) (bool, CacheRequest) {
	var caches CacheRequest
	reqType := strings.ToLower(rule.Value.Request.Type)
	method := strings.ToUpper(rule.Value.Request.Method)

	// 确保是HTTP/HTTPS请求
	if reqType != "" && reqType != common.HttpType {
		return false, caches
	}

	// 只允许GET或POST请求且header为空、body为空时使用缓存
	isEmptyHeaders := rule.Value.Request.Headers == nil || len(rule.Value.Request.Headers) == 0
	isEmptyBody := rule.Value.Request.Body == ""
	isGetOrPost := method == "GET" || method == "POST"

	if isEmptyHeaders && isEmptyBody && isGetOrPost {

		// 检查缓存中是否存在对应条目
		if target != "" {
			urlStr := common.RemoveTrailingSlash(target)
			cacheKey := GenerateCacheKey(urlStr, method, rule.Value.Request.FollowRedirects)
			logger.Debug(fmt.Sprintf("缓存提取key：%s %s %s %t", cacheKey, urlStr, method, rule.Value.Request.FollowRedirects))
			// 加读锁访问缓存
			cacheMapMux.RLock()
			entry, exists := cacheMap[cacheKey]
			cacheMapMux.RUnlock()
			if exists && entry != nil && entry.Request != nil && entry.Response != nil {
				caches.Request = entry.Request
				caches.Response = entry.Response
				return true, caches
			}
		}
	}

	return false, caches
}

// UpdateTargetCache 更新特定目标的请求响应缓存
func UpdateTargetCache(variableMap map[string]any, target string, followRedirects bool) {
	var req *proto.Request
	var resp *proto.Response

	if r, ok := variableMap["request"].(*proto.Request); ok {
		req = r
	}

	if r, ok := variableMap["response"].(*proto.Response); ok {
		resp = r
	}
	// 确保请求和响应都存在
	if req == nil || resp == nil {
		return
	}

	// 只更新结果，不需要更新缓存
	if target == "" {
		return
	}

	// 只缓存path为"/"或空、header为空、body也为空的GET或POST请求
	method := strings.ToUpper(req.Method)
	isEmptyBody := req.Body == nil || len(req.Body) == 0
	isGetOrPost := method == "GET" || method == "POST"
	if isEmptyBody && isGetOrPost {
		urlStr := common.RemoveTrailingSlash(target)
		cacheKey := GenerateCacheKey(urlStr, method, followRedirects)
		logger.Debug(fmt.Sprintf("请求缓存key：%s %s %s %t", cacheKey, urlStr, method, followRedirects))
		// 加写锁更新缓存
		cacheMapMux.Lock()
		cacheMap[cacheKey] = &CacheRequest{
			Request:  req,
			Response: resp,
		}
		cacheMapMux.Unlock()
	}
}

// ClearTargetCache 删除特定目标的请求响应缓存
func ClearTargetCache(target string, method string, followRedirects bool) {
	if target == "" {
		return
	}

	urlStr := common.RemoveTrailingSlash(target)
	cacheKey := GenerateCacheKey(urlStr, method, followRedirects)
	logger.Debug(fmt.Sprintf("删除缓存key：%s %s %s %t", cacheKey, urlStr, method, followRedirects))

	// 加写锁删除缓存
	cacheMapMux.Lock()
	delete(cacheMap, cacheKey)
	cacheMapMux.Unlock()
}

// ClearTargetURLCache 删除与特定URL相关的所有缓存，无论请求方法和跟随重定向设置如何
func ClearTargetURLCache(target string) {
	if target == "" {
		return
	}

	urlStr := common.RemoveTrailingSlash(target)
	logger.Debug(fmt.Sprintf("清除URL所有缓存：%s", urlStr))

	// 需要删除的缓存键列表
	var keysToDelete []string

	// 加读锁查找匹配的缓存键
	cacheMapMux.RLock()
	// 遍历所有缓存条目，找出包含目标URL的键
	for key := range cacheMap {
		// 这里我们不能直接通过键反向解析出URL，
		// 因为键是通过MD5哈希生成的，不可逆
		// 所以我们重新生成可能的键并比较

		// 遍历常见的HTTP方法
		methods := []string{"GET", "POST", "HEAD", "PUT", "DELETE", "OPTIONS"}
		redirectOptions := []bool{true, false}

		for _, method := range methods {
			for _, redirect := range redirectOptions {
				possibleKey := GenerateCacheKey(urlStr, method, redirect)
				if key == possibleKey {
					keysToDelete = append(keysToDelete, key)
					// 找到一个匹配就跳出内层循环
					break
				}
			}
		}
	}
	cacheMapMux.RUnlock()

	// 加写锁删除匹配的缓存条目
	if len(keysToDelete) > 0 {
		cacheMapMux.Lock()
		for _, key := range keysToDelete {
			delete(cacheMap, key)
		}
		cacheMapMux.Unlock()
		logger.Debug(fmt.Sprintf("成功删除URL相关缓存%d项：%s", len(keysToDelete), urlStr))
	} else {
		logger.Debug(fmt.Sprintf("未找到URL相关缓存：%s", urlStr))
	}
}

// ClearAllCache 清空所有缓存
func ClearAllCache() {
	cacheMapMux.Lock()
	// 重新初始化缓存映射
	cacheMap = make(map[string]*CacheRequest, 1024)
	cacheMapMux.Unlock()
	logger.Debug("已清空所有缓存")
}
