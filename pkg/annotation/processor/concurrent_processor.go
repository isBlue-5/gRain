// Package processor 提供并发处理和性能监控功能
package processor

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/isBlue-5/grain/pkg/annotation/types"
)

// ConcurrentProcessor 并发处理器
type ConcurrentProcessor struct {
	// Worker Pool
	workerPool *WorkerPool

	// 性能监控
	metrics *ProcessingMetrics

	// 进度通道
	progressChan chan ProcessingProgress

	// 配置选项
	options *ConcurrentOptions

	// 智能缓存
	smartCache *SmartBuildCache

	// 并发锁
	mu sync.RWMutex

	// 停止信号
	stopChan chan struct{}

	// 是否正在运行
	running int32
}

// WorkerPool Worker池
type WorkerPool struct {
	// Worker数量
	size int

	// 任务通道
	taskChan chan ProcessingTask

	// 结果通道
	resultChan chan ProcessingResult

	// Workers
	workers []*Worker

	// 等待组
	wg sync.WaitGroup

	// 上下文
	ctx    context.Context
	cancel context.CancelFunc
}

// Worker 工作协程
type Worker struct {
	// Worker ID
	id int

	// 任务通道
	taskChan <-chan ProcessingTask

	// 结果通道
	resultChan chan<- ProcessingResult

	// 处理器引用
	processor AnnotationProcessor

	// 统计信息
	stats *WorkerStats
}

// WorkerStats Worker统计信息
type WorkerStats struct {
	// 处理的任务数
	TasksProcessed int64

	// 总处理时间
	TotalProcessingTime time.Duration

	// 平均处理时间
	AvgProcessingTime time.Duration

	// 错误数量
	ErrorCount int64

	// 最后活动时间
	LastActiveTime time.Time
}

// ProcessingTask 处理任务
type ProcessingTask struct {
	// 任务ID
	ID string

	// 源文件路径
	SourceFile string

	// 注解列表
	Annotations []types.Annotation

	// 生成上下文
	Context *GenerationContext

	// 优先级
	Priority int

	// 创建时间
	CreatedAt time.Time

	// 重试次数
	RetryCount int
}

// ProcessingResult 处理结果
type ProcessingResult struct {
	// 任务ID
	TaskID string

	// 是否成功
	Success bool

	// 错误信息
	Error error

	// 生成的文件列表
	GeneratedFiles []string

	// 处理时间
	ProcessingTime time.Duration

	// Worker ID
	WorkerID int

	// 完成时间
	CompletedAt time.Time

	// 是否使用了缓存
	UsedCache bool
}

// ProcessingProgress 处理进度
type ProcessingProgress struct {
	// 总任务数
	TotalTasks int

	// 已完成任务数
	CompletedTasks int

	// 失败任务数
	FailedTasks int

	// 缓存命中任务数
	CachedTasks int

	// 当前处理速度 (tasks/second)
	ProcessingRate float64

	// 预计剩余时间
	EstimatedTimeRemaining time.Duration

	// 开始时间
	StartTime time.Time

	// 当前时间
	CurrentTime time.Time
}

// ProcessingMetrics 处理性能指标
type ProcessingMetrics struct {
	// 总任务数
	TotalTasks int64

	// 已完成任务数
	CompletedTasks int64

	// 失败任务数
	FailedTasks int64

	// 缓存命中数
	CacheHits int64

	// 总处理时间
	TotalProcessingTime time.Duration

	// 平均处理时间
	AvgProcessingTime time.Duration

	// 最大处理时间
	MaxProcessingTime time.Duration

	// 最小处理时间
	MinProcessingTime time.Duration

	// 吞吐量 (tasks/second)
	Throughput float64

	// 内存使用情况
	MemoryUsage *MemoryUsage

	// CPU使用情况
	CPUUsage float64

	// 开始时间
	StartTime time.Time

	// 结束时间
	EndTime time.Time

	// 锁
	mu sync.RWMutex
}

// MemoryUsage 内存使用情况
type MemoryUsage struct {
	// 当前分配的内存 (bytes)
	Alloc uint64

	// 总分配的内存 (bytes)
	TotalAlloc uint64

	// 系统内存 (bytes)
	Sys uint64

	// GC次数
	NumGC uint32

	// 上次GC时间
	LastGC time.Time
}

// ConcurrentOptions 并发处理选项
type ConcurrentOptions struct {
	// Worker数量
	WorkerCount int

	// 任务缓冲区大小
	TaskBufferSize int

	// 结果缓冲区大小
	ResultBufferSize int

	// 最大重试次数
	MaxRetries int

	// 超时时间
	Timeout time.Duration

	// 是否启用性能监控
	EnableMetrics bool

	// 是否启用进度报告
	EnableProgressReporting bool

	// 进度报告间隔
	ProgressReportInterval time.Duration

	// 是否启用自适应Worker数量
	EnableAdaptiveWorkers bool

	// CPU使用率阈值（用于自适应调整）
	CPUThreshold float64

	// 内存使用阈值 (bytes)
	MemoryThreshold uint64
}

// NewConcurrentProcessor 创建并发处理器
func NewConcurrentProcessor(cache *SmartBuildCache, options *ConcurrentOptions) *ConcurrentProcessor {
	if options == nil {
		options = DefaultConcurrentOptions()
	}

	// 如果未指定Worker数量，使用CPU核心数
	if options.WorkerCount <= 0 {
		options.WorkerCount = runtime.NumCPU()
	}

	processor := &ConcurrentProcessor{
		options:      options,
		smartCache:   cache,
		metrics:      NewProcessingMetrics(),
		progressChan: make(chan ProcessingProgress, 100),
		stopChan:     make(chan struct{}),
	}

	// 创建Worker Pool
	processor.workerPool = NewWorkerPool(options.WorkerCount, options.TaskBufferSize, options.ResultBufferSize)

	return processor
}

// DefaultConcurrentOptions 默认并发选项
func DefaultConcurrentOptions() *ConcurrentOptions {
	return &ConcurrentOptions{
		WorkerCount:             runtime.NumCPU(),
		TaskBufferSize:          1000,
		ResultBufferSize:        1000,
		MaxRetries:              3,
		Timeout:                 5 * time.Minute,
		EnableMetrics:           true,
		EnableProgressReporting: true,
		ProgressReportInterval:  1 * time.Second,
		EnableAdaptiveWorkers:   true,
		CPUThreshold:            80.0,
		MemoryThreshold:         1024 * 1024 * 1024, // 1GB
	}
}

// NewWorkerPool 创建Worker池
func NewWorkerPool(size, taskBufferSize, resultBufferSize int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		size:       size,
		taskChan:   make(chan ProcessingTask, taskBufferSize),
		resultChan: make(chan ProcessingResult, resultBufferSize),
		workers:    make([]*Worker, 0, size),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// NewProcessingMetrics 创建处理指标
func NewProcessingMetrics() *ProcessingMetrics {
	return &ProcessingMetrics{
		MinProcessingTime: time.Duration(^uint64(0) >> 1), // 最大值
		MemoryUsage:       &MemoryUsage{},
		StartTime:         time.Now(),
	}
}

// Start 启动并发处理器
func (cp *ConcurrentProcessor) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&cp.running, 0, 1) {
		return fmt.Errorf("处理器已经在运行")
	}

	cp.mu.Lock()
	defer cp.mu.Unlock()

	// 启动Worker Pool
	if err := cp.workerPool.Start(ctx); err != nil {
		atomic.StoreInt32(&cp.running, 0)
		return fmt.Errorf("启动Worker Pool失败: %w", err)
	}

	// 启动结果处理协程
	go cp.handleResults(ctx)

	// 启动性能监控
	if cp.options.EnableMetrics {
		go cp.monitorPerformance(ctx)
	}

	// 启动进度报告
	if cp.options.EnableProgressReporting {
		go cp.reportProgress(ctx)
	}

	// 启动自适应Worker调整
	if cp.options.EnableAdaptiveWorkers {
		go cp.adaptiveWorkerAdjustment(ctx)
	}

	return nil
}

// Stop 停止并发处理器
func (cp *ConcurrentProcessor) Stop() error {
	if !atomic.CompareAndSwapInt32(&cp.running, 1, 0) {
		return fmt.Errorf("处理器未运行")
	}

	cp.mu.Lock()
	defer cp.mu.Unlock()

	// 发送停止信号
	close(cp.stopChan)

	// 停止Worker Pool
	if err := cp.workerPool.Stop(); err != nil {
		return fmt.Errorf("停止Worker Pool失败: %w", err)
	}

	// 记录结束时间
	cp.metrics.EndTime = time.Now()

	return nil
}

// ProcessBatch 批量处理任务
func (cp *ConcurrentProcessor) ProcessBatch(ctx context.Context, tasks []ProcessingTask) (<-chan ProcessingResult, error) {
	if atomic.LoadInt32(&cp.running) != 1 {
		return nil, fmt.Errorf("处理器未运行")
	}

	// 更新总任务数
	cp.metrics.mu.Lock()
	cp.metrics.TotalTasks += int64(len(tasks))
	cp.metrics.mu.Unlock()

	// 提交任务到Worker Pool
	resultChan := make(chan ProcessingResult, len(tasks))

	go func() {
		defer close(resultChan)

		// 提交所有任务
		for _, task := range tasks {
			select {
			case cp.workerPool.taskChan <- task:
				// 任务提交成功
			case <-ctx.Done():
				return
			case <-cp.stopChan:
				return
			}
		}

		// 等待所有结果
		completedCount := 0
		for completedCount < len(tasks) {
			select {
			case result := <-cp.workerPool.resultChan:
				resultChan <- result
				completedCount++
			case <-ctx.Done():
				return
			case <-cp.stopChan:
				return
			}
		}
	}()

	return resultChan, nil
}

// Start 启动Worker Pool
func (wp *WorkerPool) Start(ctx context.Context) error {
	// 创建并启动Workers
	for i := 0; i < wp.size; i++ {
		worker := &Worker{
			id:         i,
			taskChan:   wp.taskChan,
			resultChan: wp.resultChan,
			stats:      &WorkerStats{},
		}

		wp.workers = append(wp.workers, worker)

		wp.wg.Add(1)
		go worker.run(wp.ctx, &wp.wg)
	}

	return nil
}

// Stop 停止Worker Pool
func (wp *WorkerPool) Stop() error {
	// 取消上下文
	wp.cancel()

	// 关闭任务通道
	close(wp.taskChan)

	// 等待所有Worker完成
	wp.wg.Wait()

	// 关闭结果通道
	close(wp.resultChan)

	return nil
}

// run Worker运行循环
func (w *Worker) run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case task, ok := <-w.taskChan:
			if !ok {
				return // 通道已关闭
			}

			// 处理任务
			result := w.processTask(task)

			// 发送结果
			select {
			case w.resultChan <- result:
			case <-ctx.Done():
				return
			}

		case <-ctx.Done():
			return
		}
	}
}

// processTask 处理单个任务
func (w *Worker) processTask(task ProcessingTask) ProcessingResult {
	startTime := time.Now()
	w.stats.LastActiveTime = startTime

	result := ProcessingResult{
		TaskID:      task.ID,
		WorkerID:    w.id,
		CompletedAt: startTime,
	}

	// TODO: 实际的任务处理逻辑
	// 这里应该调用具体的注解处理器

	// 模拟处理时间
	time.Sleep(10 * time.Millisecond)

	processingTime := time.Since(startTime)
	result.ProcessingTime = processingTime
	result.Success = true // 简化实现

	// 更新Worker统计
	atomic.AddInt64(&w.stats.TasksProcessed, 1)
	w.stats.TotalProcessingTime += processingTime
	w.stats.AvgProcessingTime = time.Duration(int64(w.stats.TotalProcessingTime) / w.stats.TasksProcessed)

	return result
}

// handleResults 处理结果
func (cp *ConcurrentProcessor) handleResults(ctx context.Context) {
	for {
		select {
		case result := <-cp.workerPool.resultChan:
			cp.updateMetrics(result)

		case <-ctx.Done():
			return
		case <-cp.stopChan:
			return
		}
	}
}

// updateMetrics 更新性能指标
func (cp *ConcurrentProcessor) updateMetrics(result ProcessingResult) {
	cp.metrics.mu.Lock()
	defer cp.metrics.mu.Unlock()

	if result.Success {
		cp.metrics.CompletedTasks++
		if result.UsedCache {
			cp.metrics.CacheHits++
		}
	} else {
		cp.metrics.FailedTasks++
	}

	// 更新处理时间统计
	cp.metrics.TotalProcessingTime += result.ProcessingTime

	if cp.metrics.CompletedTasks > 0 {
		cp.metrics.AvgProcessingTime = time.Duration(int64(cp.metrics.TotalProcessingTime) / cp.metrics.CompletedTasks)
	}

	if result.ProcessingTime > cp.metrics.MaxProcessingTime {
		cp.metrics.MaxProcessingTime = result.ProcessingTime
	}

	if result.ProcessingTime < cp.metrics.MinProcessingTime {
		cp.metrics.MinProcessingTime = result.ProcessingTime
	}

	// 计算吞吐量
	elapsed := time.Since(cp.metrics.StartTime)
	if elapsed > 0 {
		cp.metrics.Throughput = float64(cp.metrics.CompletedTasks) / elapsed.Seconds()
	}
}

// monitorPerformance 监控性能
func (cp *ConcurrentProcessor) monitorPerformance(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cp.collectSystemMetrics()

		case <-ctx.Done():
			return
		case <-cp.stopChan:
			return
		}
	}
}

// collectSystemMetrics 收集系统指标
func (cp *ConcurrentProcessor) collectSystemMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	cp.metrics.mu.Lock()
	defer cp.metrics.mu.Unlock()

	cp.metrics.MemoryUsage.Alloc = m.Alloc
	cp.metrics.MemoryUsage.TotalAlloc = m.TotalAlloc
	cp.metrics.MemoryUsage.Sys = m.Sys
	cp.metrics.MemoryUsage.NumGC = m.NumGC

	if m.LastGC > 0 {
		cp.metrics.MemoryUsage.LastGC = time.Unix(0, int64(m.LastGC))
	}

	// TODO: 收集CPU使用率
	// 这需要额外的系统调用或第三方库
}

// reportProgress 报告进度
func (cp *ConcurrentProcessor) reportProgress(ctx context.Context) {
	ticker := time.NewTicker(cp.options.ProgressReportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			progress := cp.calculateProgress()

			select {
			case cp.progressChan <- progress:
			default:
				// 进度通道满了，跳过这次报告
			}

		case <-ctx.Done():
			return
		case <-cp.stopChan:
			return
		}
	}
}

// calculateProgress 计算进度
func (cp *ConcurrentProcessor) calculateProgress() ProcessingProgress {
	cp.metrics.mu.RLock()
	defer cp.metrics.mu.RUnlock()

	progress := ProcessingProgress{
		TotalTasks:     int(cp.metrics.TotalTasks),
		CompletedTasks: int(cp.metrics.CompletedTasks),
		FailedTasks:    int(cp.metrics.FailedTasks),
		CachedTasks:    int(cp.metrics.CacheHits),
		StartTime:      cp.metrics.StartTime,
		CurrentTime:    time.Now(),
	}

	// 计算处理速度
	elapsed := progress.CurrentTime.Sub(progress.StartTime)
	if elapsed > 0 {
		progress.ProcessingRate = float64(progress.CompletedTasks) / elapsed.Seconds()
	}

	// 估算剩余时间
	if progress.ProcessingRate > 0 {
		remainingTasks := progress.TotalTasks - progress.CompletedTasks
		progress.EstimatedTimeRemaining = time.Duration(float64(remainingTasks)/progress.ProcessingRate) * time.Second
	}

	return progress
}

// adaptiveWorkerAdjustment 自适应Worker数量调整
func (cp *ConcurrentProcessor) adaptiveWorkerAdjustment(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cp.adjustWorkerCount()

		case <-ctx.Done():
			return
		case <-cp.stopChan:
			return
		}
	}
}

// adjustWorkerCount 调整Worker数量
func (cp *ConcurrentProcessor) adjustWorkerCount() {
	cp.metrics.mu.RLock()
	cpuUsage := cp.metrics.CPUUsage
	memUsage := cp.metrics.MemoryUsage.Alloc
	cp.metrics.mu.RUnlock()

	currentWorkerCount := len(cp.workerPool.workers)

	// 如果CPU使用率过高，减少Worker
	if cpuUsage > cp.options.CPUThreshold && currentWorkerCount > 1 {
		// TODO: 实现动态减少Worker的逻辑
		return
	}

	// 如果内存使用过高，减少Worker
	if memUsage > cp.options.MemoryThreshold && currentWorkerCount > 1 {
		// TODO: 实现动态减少Worker的逻辑
		return
	}

	// 如果资源使用率较低且有待处理任务，增加Worker
	if cpuUsage < cp.options.CPUThreshold*0.7 && memUsage < cp.options.MemoryThreshold*0.7 {
		// 检查任务队列长度
		if len(cp.workerPool.taskChan) > currentWorkerCount*2 {
			// TODO: 实现动态增加Worker的逻辑
		}
	}
}

// GetProgressChannel 获取进度通道
func (cp *ConcurrentProcessor) GetProgressChannel() <-chan ProcessingProgress {
	return cp.progressChan
}

// GetMetrics 获取性能指标
func (cp *ConcurrentProcessor) GetMetrics() *ProcessingMetrics {
	cp.metrics.mu.RLock()
	defer cp.metrics.mu.RUnlock()

	// 返回副本以避免并发修改
	metrics := *cp.metrics
	memUsage := *cp.metrics.MemoryUsage
	metrics.MemoryUsage = &memUsage

	return &metrics
}

// GetWorkerStats 获取Worker统计信息
func (cp *ConcurrentProcessor) GetWorkerStats() []*WorkerStats {
	stats := make([]*WorkerStats, len(cp.workerPool.workers))

	for i, worker := range cp.workerPool.workers {
		// 创建副本
		stat := *worker.stats
		stats[i] = &stat
	}

	return stats
}
