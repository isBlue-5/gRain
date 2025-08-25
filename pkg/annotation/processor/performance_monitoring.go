// Package processor provides performance monitoring capabilities
package processor

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"time"
)

// PerformanceMonitor 性能监控器
type PerformanceMonitor struct {
	metrics map[string]*PerformanceMetric
	mu      sync.RWMutex
	enabled bool
}

// PerformanceMetric 性能指标
type PerformanceMetric struct {
	Count      int64         `json:"count"`
	TotalTime  time.Duration `json:"total_time"`
	MinTime    time.Duration `json:"min_time"`
	MaxTime    time.Duration `json:"max_time"`
	AvgTime    time.Duration `json:"avg_time"`
	Operations []string      `json:"operations"`
	LastUpdate time.Time     `json:"last_update"`
}

// NewPerformanceMonitor 创建新的性能监控器
func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		metrics: make(map[string]*PerformanceMetric),
		enabled: true,
	}
}

// Record 记录操作性能
func (pm *PerformanceMonitor) Record(operation string, duration time.Duration, details ...string) {
	if !pm.enabled {
		return
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	metric, exists := pm.metrics[operation]
	if !exists {
		metric = &PerformanceMetric{
			Operations: make([]string, 0),
		}
		pm.metrics[operation] = metric
	}

	metric.Count++
	metric.TotalTime += duration
	metric.LastUpdate = time.Now()

	// 更新最小和最大时间
	if metric.MinTime == 0 || duration < metric.MinTime {
		metric.MinTime = duration
	}
	if duration > metric.MaxTime {
		metric.MaxTime = duration
	}

	// 计算平均时间
	metric.AvgTime = metric.TotalTime / time.Duration(metric.Count)

	// 记录操作详情（最多保留最新的10个）
	if len(details) > 0 {
		metric.Operations = append(metric.Operations, details[0])
		if len(metric.Operations) > 10 {
			metric.Operations = metric.Operations[1:]
		}
	}
}

// StartTimer 开始计时器
func (pm *PerformanceMonitor) StartTimer(operation string) *Timer {
	return &Timer{
		monitor:   pm,
		operation: operation,
		start:     time.Now(),
	}
}

// GetMetrics 获取所有性能指标
func (pm *PerformanceMonitor) GetMetrics() map[string]*PerformanceMetric {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// 返回副本以避免并发修改
	result := make(map[string]*PerformanceMetric)
	for k, v := range pm.metrics {
		result[k] = &PerformanceMetric{
			Count:      v.Count,
			TotalTime:  v.TotalTime,
			MinTime:    v.MinTime,
			MaxTime:    v.MaxTime,
			AvgTime:    v.AvgTime,
			Operations: append([]string{}, v.Operations...),
			LastUpdate: v.LastUpdate,
		}
	}

	return result
}

// GetMetric 获取特定操作的性能指标
func (pm *PerformanceMonitor) GetMetric(operation string) *PerformanceMetric {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if metric, exists := pm.metrics[operation]; exists {
		return &PerformanceMetric{
			Count:      metric.Count,
			TotalTime:  metric.TotalTime,
			MinTime:    metric.MinTime,
			MaxTime:    metric.MaxTime,
			AvgTime:    metric.AvgTime,
			Operations: append([]string{}, metric.Operations...),
			LastUpdate: metric.LastUpdate,
		}
	}

	return nil
}

// Reset 重置所有性能指标
func (pm *PerformanceMonitor) Reset() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.metrics = make(map[string]*PerformanceMetric)
}

// Enable 启用性能监控
func (pm *PerformanceMonitor) Enable() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.enabled = true
}

// Disable 禁用性能监控
func (pm *PerformanceMonitor) Disable() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.enabled = false
}

// GenerateReport 生成性能报告
func (pm *PerformanceMonitor) GenerateReport() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	report := "=== Performance Report ===\n\n"

	// 添加系统信息
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	report += fmt.Sprintf("System Info:\n")
	report += fmt.Sprintf("  - Go Routines: %d\n", runtime.NumGoroutine())
	report += fmt.Sprintf("  - Memory Alloc: %d KB\n", m.Alloc/1024)
	report += fmt.Sprintf("  - Memory Total Alloc: %d KB\n", m.TotalAlloc/1024)
	report += fmt.Sprintf("  - Memory Sys: %d KB\n", m.Sys/1024)
	report += fmt.Sprintf("  - GC Cycles: %d\n\n", m.NumGC)

	// 添加性能指标
	if len(pm.metrics) == 0 {
		report += "No performance metrics recorded.\n"
		return report
	}

	report += "Performance Metrics:\n"
	for operation, metric := range pm.metrics {
		report += fmt.Sprintf("\n%s:\n", operation)
		report += fmt.Sprintf("  - Count: %d\n", metric.Count)
		report += fmt.Sprintf("  - Total Time: %v\n", metric.TotalTime)
		report += fmt.Sprintf("  - Average Time: %v\n", metric.AvgTime)
		report += fmt.Sprintf("  - Min Time: %v\n", metric.MinTime)
		report += fmt.Sprintf("  - Max Time: %v\n", metric.MaxTime)
		report += fmt.Sprintf("  - Last Update: %v\n", metric.LastUpdate.Format("2006-01-02 15:04:05"))

		if len(metric.Operations) > 0 {
			report += "  - Recent Operations:\n"
			for _, op := range metric.Operations {
				report += fmt.Sprintf("    * %s\n", op)
			}
		}
	}

	return report
}

// Timer 性能计时器
type Timer struct {
	monitor   *PerformanceMonitor
	operation string
	start     time.Time
	details   []string
}

// Stop 停止计时器并记录性能
func (t *Timer) Stop(details ...string) time.Duration {
	duration := time.Since(t.start)
	t.monitor.Record(t.operation, duration, details...)
	return duration
}

// AddDetail 添加详细信息
func (t *Timer) AddDetail(detail string) {
	t.details = append(t.details, detail)
}

// GlobalPerformanceMonitor 全局性能监控器
var GlobalPerformanceMonitor = NewPerformanceMonitor()

// RecordOperation 记录操作性能（便捷方法）
func RecordOperation(operation string, fn func() error, details ...string) error {
	timer := GlobalPerformanceMonitor.StartTimer(operation)
	defer timer.Stop(details...)

	return fn()
}

// BenchmarkProcessor 处理器性能基准测试
type BenchmarkProcessor struct {
	monitor *PerformanceMonitor
	logger  *log.Logger
}

// NewBenchmarkProcessor 创建新的基准测试处理器
func NewBenchmarkProcessor() *BenchmarkProcessor {
	return &BenchmarkProcessor{
		monitor: NewPerformanceMonitor(),
		logger:  log.New(os.Stdout, "[Benchmark] ", log.LstdFlags),
	}
}

// BenchmarkCodeGeneration 基准测试代码生成性能
func (bp *BenchmarkProcessor) BenchmarkCodeGeneration(files []string, iterations int) *PerformanceReport {
	report := &PerformanceReport{
		TestName:   "Code Generation Benchmark",
		StartTime:  time.Now(),
		Iterations: iterations,
		Files:      len(files),
	}

	// 预热
	for i := 0; i < 3; i++ {
		bp.runCodeGeneration(files)
	}

	// 实际测试
	var totalDuration time.Duration
	durations := make([]time.Duration, iterations)

	for i := 0; i < iterations; i++ {
		start := time.Now()
		bp.runCodeGeneration(files)
		duration := time.Since(start)

		durations[i] = duration
		totalDuration += duration
	}

	report.EndTime = time.Now()
	report.TotalDuration = totalDuration
	report.AverageDuration = totalDuration / time.Duration(iterations)

	// 计算最小和最大时间
	for _, d := range durations {
		if report.MinDuration == 0 || d < report.MinDuration {
			report.MinDuration = d
		}
		if d > report.MaxDuration {
			report.MaxDuration = d
		}
	}

	return report
}

// runCodeGeneration 运行代码生成（模拟）
func (bp *BenchmarkProcessor) runCodeGeneration(files []string) {
	// 这里应该调用实际的代码生成逻辑
	// 为了测试，我们模拟一些处理时间

	// 创建代码生成器
	generator := NewCodeGenerator()

	// 处理每个文件
	for _, file := range files {
		start := time.Now()

		// 解析文件
		if err := generator.ParseFile(file); err != nil {
			bp.logger.Printf("Failed to parse file %s: %v", file, err)
			continue
		}

		// 生成代码
		if err := generator.GenerateCode(file); err != nil {
			bp.logger.Printf("Failed to generate code for %s: %v", file, err)
			continue
		}

		// 记录处理时间
		duration := time.Since(start)
		bp.logger.Printf("Processed %s in %v", file, duration)
	}
}

// NewCodeGenerator 创建代码生成器
func NewCodeGenerator() *CodeGenerator {
	return &CodeGenerator{}
}

// CodeGenerator 代码生成器
type CodeGenerator struct{}

// ParseFile 解析文件
func (cg *CodeGenerator) ParseFile(filename string) error {
	// 实现文件解析逻辑
	return nil
}

// GenerateCode 生成代码
func (cg *CodeGenerator) GenerateCode(filename string) error {
	// 实现代码生成逻辑
	return nil
}

// PerformanceReport 性能报告
type PerformanceReport struct {
	TestName        string        `json:"test_name"`
	StartTime       time.Time     `json:"start_time"`
	EndTime         time.Time     `json:"end_time"`
	TotalDuration   time.Duration `json:"total_duration"`
	AverageDuration time.Duration `json:"average_duration"`
	MinDuration     time.Duration `json:"min_duration"`
	MaxDuration     time.Duration `json:"max_duration"`
	Iterations      int           `json:"iterations"`
	Files           int           `json:"files"`
}

// String 返回报告的字符串表示
func (pr *PerformanceReport) String() string {
	return fmt.Sprintf(`
Performance Report: %s
=====================================
Test Period: %s - %s
Total Duration: %v
Average Duration: %v
Min Duration: %v
Max Duration: %v
Iterations: %d
Files Processed: %d
Throughput: %.2f files/second
`, pr.TestName, pr.StartTime.Format("15:04:05"), pr.EndTime.Format("15:04:05"),
		pr.TotalDuration, pr.AverageDuration, pr.MinDuration, pr.MaxDuration,
		pr.Iterations, pr.Files, float64(pr.Files*pr.Iterations)/pr.TotalDuration.Seconds())
}

// MemoryProfiler 内存分析器
type MemoryProfiler struct {
	initialStats runtime.MemStats
	enabled      bool
}

// NewMemoryProfiler 创建新的内存分析器
func NewMemoryProfiler() *MemoryProfiler {
	profiler := &MemoryProfiler{enabled: true}
	runtime.ReadMemStats(&profiler.initialStats)
	return profiler
}

// Profile 分析内存使用情况
func (mp *MemoryProfiler) Profile(operation string, fn func()) *MemoryReport {
	if !mp.enabled {
		fn()
		return &MemoryReport{}
	}

	// 强制GC以获得更准确的内存统计
	runtime.GC()

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)

	start := time.Now()
	fn()
	duration := time.Since(start)

	runtime.ReadMemStats(&after)

	return &MemoryReport{
		Operation:    operation,
		Duration:     duration,
		AllocBefore:  before.Alloc,
		AllocAfter:   after.Alloc,
		AllocDelta:   int64(after.Alloc) - int64(before.Alloc),
		TotalAlloc:   after.TotalAlloc - before.TotalAlloc,
		NumGC:        after.NumGC - before.NumGC,
		NumGoroutine: runtime.NumGoroutine(),
	}
}

// MemoryReport 内存使用报告
type MemoryReport struct {
	Operation    string        `json:"operation"`
	Duration     time.Duration `json:"duration"`
	AllocBefore  uint64        `json:"alloc_before"`
	AllocAfter   uint64        `json:"alloc_after"`
	AllocDelta   int64         `json:"alloc_delta"`
	TotalAlloc   uint64        `json:"total_alloc"`
	NumGC        uint32        `json:"num_gc"`
	NumGoroutine int           `json:"num_goroutine"`
}

// String 返回内存报告的字符串表示
func (mr *MemoryReport) String() string {
	return fmt.Sprintf(`
Memory Report: %s
=============================
Duration: %v
Memory Before: %s
Memory After: %s
Memory Delta: %s
Total Allocated: %s
GC Cycles: %d
Goroutines: %d
`, mr.Operation, mr.Duration,
		formatBytes(mr.AllocBefore),
		formatBytes(mr.AllocAfter),
		formatBytesDelta(mr.AllocDelta),
		formatBytes(mr.TotalAlloc),
		mr.NumGC, mr.NumGoroutine)
}

// formatBytes 格式化字节数
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatBytesDelta 格式化字节数差值
func formatBytesDelta(delta int64) string {
	sign := ""
	if delta > 0 {
		sign = "+"
	}
	return sign + formatBytes(uint64(abs(delta)))
}

// abs 返回绝对值
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
