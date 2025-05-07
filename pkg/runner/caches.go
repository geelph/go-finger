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
)

// CacheRequest 存储请求和响应的缓存条目
type CacheRequest struct {
	Request  *proto.Request
	Response *proto.Response
}

// CacheShard 缓存分片，每个分片有自己的锁和映射
type CacheShard struct {
	items map[string]*CacheRequest
}

const (
	// ShardCount 分片数量，使用2的幂次方便于位运算
	ShardCount = 32
	// ShardMask 分片掩码
	ShardMask = ShardCount - 1
)

// 分片缓存，减少锁竞争
var cacheShards [ShardCount]*CacheShard

// 初始化分片缓存
func init() {
	for i := 0; i < ShardCount; i++ {
		cacheShards[i] = &CacheShard{
			items: make(map[string]*CacheRequest, 256), // 预分配空间以减少哈希表扩容
		}
	}
}

// getShard 根据key获取对应的分片
func getShard(key string) *CacheShard {
	// 简单的哈希函数，将key映射到分片索引
	hash := 0
	for i := 0; i < len(key); i++ {
		hash = 31*hash + int(key[i])
	}
	return cacheShards[hash&ShardMask]
}

// GenerateCacheKey 生成缓存键
func GenerateCacheKey(target string, method string) string {
	return target + ":" + method
}

// ShouldUseCache 判断是否应该使用缓存，对于根路径的GET请求，可以重用缓存的请求和响应
func ShouldUseCache(rule finger.RuleMap, targetResult *TargetResult) (bool, CacheRequest) {
	var caches CacheRequest
	reqType := strings.ToLower(rule.Value.Request.Type)
	method := strings.ToUpper(rule.Value.Request.Method)
	// 只允许GET请求或空body的POST请求使用缓存
	if !(method == "GET" || (method == "POST" && rule.Value.Request.Body == "")) && rule.Value.Request.FollowRedirects != false {
		return false, caches
	}

	// 确保是HTTP/HTTPS请求
	if reqType != "" && reqType != common.HttpType {
		return false, caches
	}

	// 检查缓存中是否存在对应条目
	if targetResult.URL != "" {
		cacheKey := GenerateCacheKey(targetResult.URL, method)

		// 获取对应的分片并读取缓存
		shard := getShard(cacheKey)
		entry, exists := shard.items[cacheKey]

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

	// 只缓存GET请求或空body的POST请求
	method := strings.ToUpper(req.Method)
	if method == "GET" || (method == "POST" && (req.Body == nil || len(req.Body) == 0)) {
		cacheKey := GenerateCacheKey(targetResult.URL, method)

		// 获取对应的分片并更新缓存
		shard := getShard(cacheKey)
		shard.items[cacheKey] = &CacheRequest{
			Request:  req,
			Response: resp,
		}
	}
}
