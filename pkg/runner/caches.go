/*
  - Package caches
    @Author: zhizhuo
    @IDE：GoLand
    @File: caches.go
    @Date: 2025/5/6 上午11:39*
*/
package runner

import (
	"gxx/pkg/finger"
	"gxx/utils/common"
	"gxx/utils/proto"
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
func GenerateCacheKey(target string, method string) string {
	return target + ":" + method
}

// ShouldUseCache 判断是否应该使用缓存，对于根路径的GET请求，可以重用缓存的请求和响应
func ShouldUseCache(rule finger.RuleMap, targetResult *TargetResult) (bool, CacheRequest) {
	var caches CacheRequest
	reqType := strings.ToLower(rule.Value.Request.Type)
	method := strings.ToUpper(rule.Value.Request.Method)

	// 确保是HTTP/HTTPS请求
	if reqType != "" && reqType != common.HttpType {
		return false, caches
	}

	// 只允许GET或POST请求且path为"/"或空、header为空、body为空时使用缓存
	isEmptyPath := rule.Value.Request.Path == "/" || rule.Value.Request.Path == ""
	isEmptyHeaders := rule.Value.Request.Headers == nil || len(rule.Value.Request.Headers) == 0
	isEmptyBody := rule.Value.Request.Body == ""
	isGetOrPost := method == "GET" || method == "POST"

	if !(isEmptyPath && isEmptyHeaders && isEmptyBody && isGetOrPost) || rule.Value.Request.FollowRedirects != false {
		return false, caches
	}

	// 检查缓存中是否存在对应条目
	if targetResult.URL != "" {
		cacheKey := GenerateCacheKey(targetResult.URL, method)

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
	return false, caches
}

// UpdateTargetCache 更新特定目标的请求响应缓存
func UpdateTargetCache(variableMap map[string]any, targetResult *TargetResult) {
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
	if targetResult.URL == "" {
		return
	}

	// 只缓存path为"/"或空、header为空、body也为空的GET或POST请求
	method := strings.ToUpper(req.Method)
	isEmptyPath := req.Url.Path == "/" || len(req.Url.Path) == 0
	isEmptyBody := req.Body == nil || len(req.Body) == 0
	isGetOrPost := method == "GET" || method == "POST"

	if isEmptyPath && isEmptyBody && isGetOrPost {
		cacheKey := GenerateCacheKey(targetResult.URL, method)
		// 加写锁更新缓存
		cacheMapMux.Lock()
		cacheMap[cacheKey] = &CacheRequest{
			Request:  req,
			Response: resp,
		}
		cacheMapMux.Unlock()
	}
}
