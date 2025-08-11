package runner

import (
	"fmt"
	"gxx/utils/logger"
	"runtime"
	"runtime/debug"
	"sync/atomic"
	"time"
)

// MemoryStats 内存统计信息
type MemoryStats struct {
	HeapAlloc     uint64    // 堆已分配内存 (字节)
	HeapSys       uint64    // 堆系统内存 (字节)
	NumGC         uint32    // GC次数
	GCCPUFraction float64   // GC占用CPU时间比例
	LastGCTime    time.Time // 上次GC时间
	MemoryUsage   float64   // 内存使用率 (%)
}

// PerformanceMonitor 性能监控器
type PerformanceMonitor struct {
	enabled              atomic.Bool
	lastGCTime           int64
	highMemThreshold     uint64 // 高内存使用阈值
	criticalMemThreshold uint64 // 临界内存使用阈值
}

var globalMonitor *PerformanceMonitor

// 初始化全局监控器
func init() {
	globalMonitor = &PerformanceMonitor{
		highMemThreshold:     2 * 1024 * 1024 * 1024, // 2GB
		criticalMemThreshold: 4 * 1024 * 1024 * 1024, // 4GB
	}
}

// StartMemoryMonitor 启动内存监控
func StartMemoryMonitor() {
	if !globalMonitor.enabled.CompareAndSwap(false, true) {
		return // 已经启动
	}

	go globalMonitor.monitorLoop()
	logger.Info("内存监控已启动")
}

// StopMemoryMonitor 停止内存监控
func StopMemoryMonitor() {
	globalMonitor.enabled.Store(false)
	logger.Info("内存监控已停止")
}

// monitorLoop 监控循环
func (pm *PerformanceMonitor) monitorLoop() {
	ticker := time.NewTicker(30 * time.Second) // 30秒检查一次
	defer ticker.Stop()

	for pm.enabled.Load() {
		select {
		case <-ticker.C:
			pm.checkMemoryUsage()
		}
	}
}

// checkMemoryUsage 检查内存使用情况
func (pm *PerformanceMonitor) checkMemoryUsage() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	stats := MemoryStats{
		HeapAlloc:     memStats.HeapAlloc,
		HeapSys:       memStats.HeapSys,
		NumGC:         memStats.NumGC,
		GCCPUFraction: memStats.GCCPUFraction,
		LastGCTime:    time.Unix(0, int64(memStats.LastGC)),
		MemoryUsage:   float64(memStats.HeapAlloc) / float64(memStats.HeapSys) * 100,
	}

	// 记录内存使用情况
	logger.Debug(fmt.Sprintf("内存使用: %.2f MB (%.1f%%), GC次数: %d",
		float64(stats.HeapAlloc)/1024/1024, stats.MemoryUsage, stats.NumGC))

	// 根据内存使用情况采取措施
	pm.handleMemoryPressure(&stats)
}

// handleMemoryPressure 处理内存压力
func (pm *PerformanceMonitor) handleMemoryPressure(stats *MemoryStats) {
	// 检查是否需要强制GC
	shouldForceGC := false

	// 条件1: 内存使用超过高阈值
	if stats.HeapAlloc > pm.highMemThreshold {
		shouldForceGC = true
		logger.Debug("内存使用超过高阈值，触发GC")
	}

	// 条件2: 内存使用率超过85%
	if stats.MemoryUsage > 85.0 {
		shouldForceGC = true
		logger.Debug("内存使用率过高，触发GC")
	}

	// 条件3: 距离上次GC时间超过2分钟
	if time.Since(stats.LastGCTime) > 2*time.Minute {
		shouldForceGC = true
		logger.Debug("距离上次GC时间过长，触发GC")
	}

	if shouldForceGC {
		runtime.GC()

		// 如果内存使用仍然很高，释放系统内存
		if stats.HeapAlloc > pm.criticalMemThreshold {
			logger.Debug("内存使用达到临界值，释放系统内存")
			debug.FreeOSMemory()
		}
	}
}

// GetMemoryStats 获取当前内存统计信息
func GetMemoryStats() MemoryStats {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return MemoryStats{
		HeapAlloc:     memStats.HeapAlloc,
		HeapSys:       memStats.HeapSys,
		NumGC:         memStats.NumGC,
		GCCPUFraction: memStats.GCCPUFraction,
		LastGCTime:    time.Unix(0, int64(memStats.LastGC)),
		MemoryUsage:   float64(memStats.HeapAlloc) / float64(memStats.HeapSys) * 100,
	}
}

// ForceGC 强制执行垃圾回收
func ForceGC() {
	before := GetMemoryStats()
	runtime.GC()
	after := GetMemoryStats()

	logger.Debug(fmt.Sprintf("强制GC执行完成，内存释放: %.2f MB",
		float64(before.HeapAlloc-after.HeapAlloc)/1024/1024))
}

// SetMemoryThresholds 设置内存阈值
func SetMemoryThresholds(highThreshold, criticalThreshold uint64) {
	globalMonitor.highMemThreshold = highThreshold
	globalMonitor.criticalMemThreshold = criticalThreshold
	logger.Info(fmt.Sprintf("内存阈值已更新: 高阈值=%.2f MB, 临界阈值=%.2f MB",
		float64(highThreshold)/1024/1024, float64(criticalThreshold)/1024/1024))
}
