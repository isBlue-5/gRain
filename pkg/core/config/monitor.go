package config

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// ConfigMonitor 配置监控器
type ConfigMonitor struct {
	// 监控的文件
	watchedFiles map[string]*FileWatcher

	// 重载间隔
	reloadInterval time.Duration

	// 监控锁
	mu sync.RWMutex

	// 重载计数
	reloadCount int

	// 停止信号
	stopChan chan struct{}
}

// FileWatcher 文件监控器
type FileWatcher struct {
	// 文件路径
	Path string

	// 最后修改时间
	LastModTime time.Time

	// 文件大小
	LastSize int64

	// 回调函数
	Callback func(interface{}) error

	// 停止信号
	stopChan chan struct{}
}

// NewConfigMonitor 创建新的配置监控器
func NewConfigMonitor(reloadInterval time.Duration) *ConfigMonitor {
	return &ConfigMonitor{
		watchedFiles:   make(map[string]*FileWatcher),
		reloadInterval: reloadInterval,
		stopChan:       make(chan struct{}),
	}
}

// WatchFile 监控文件变化
func (cm *ConfigMonitor) WatchFile(configPath string, callback func(interface{}) error) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 检查文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("config file does not exist: %s", configPath)
	}

	// 获取文件信息
	fileInfo, err := os.Stat(configPath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %s", err)
	}

	// 创建文件监控器
	watcher := &FileWatcher{
		Path:        configPath,
		LastModTime: fileInfo.ModTime(),
		LastSize:    fileInfo.Size(),
		Callback:    callback,
		stopChan:    make(chan struct{}),
	}

	// 添加到监控列表
	cm.watchedFiles[configPath] = watcher

	// 启动监控协程
	go cm.watchFile(watcher)

	return nil
}

// watchFile 监控单个文件
func (cm *ConfigMonitor) watchFile(watcher *FileWatcher) {
	ticker := time.NewTicker(cm.reloadInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := cm.checkFileChanges(watcher); err != nil {
				// 记录错误但不停止监控
				fmt.Printf("Error checking file changes for %s: %v\n", watcher.Path, err)
			}
		case <-watcher.stopChan:
			return
		case <-cm.stopChan:
			return
		}
	}
}

// checkFileChanges 检查文件变化
func (cm *ConfigMonitor) checkFileChanges(watcher *FileWatcher) error {
	// 获取当前文件信息
	fileInfo, err := os.Stat(watcher.Path)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// 检查文件是否发生变化
	if fileInfo.ModTime().Equal(watcher.LastModTime) && fileInfo.Size() == watcher.LastSize {
		return nil // 文件没有变化
	}

	// 文件发生变化，更新监控器状态
	watcher.LastModTime = fileInfo.ModTime()
	watcher.LastSize = fileInfo.Size()

	// 增加重载计数
	cm.mu.Lock()
	cm.reloadCount++
	cm.mu.Unlock()

	// 调用回调函数
	if watcher.Callback != nil {
		if err := watcher.Callback(nil); err != nil {
			return fmt.Errorf("callback execution failed: %w", err)
		}
	}

	return nil
}

// UnwatchFile 停止监控文件
func (cm *ConfigMonitor) UnwatchFile(configPath string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	watcher, exists := cm.watchedFiles[configPath]
	if !exists {
		return fmt.Errorf("file %s is not being watched", configPath)
	}

	// 发送停止信号
	close(watcher.stopChan)

	// 从监控列表中移除
	delete(cm.watchedFiles, configPath)

	return nil
}

// UnwatchAll 停止监控所有文件
func (cm *ConfigMonitor) UnwatchAll() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 发送停止信号给所有监控器
	for _, watcher := range cm.watchedFiles {
		close(watcher.stopChan)
	}

	// 清空监控列表
	cm.watchedFiles = make(map[string]*FileWatcher)
}

// Stop 停止监控器
func (cm *ConfigMonitor) Stop() {
	// 停止所有文件监控
	cm.UnwatchAll()

	// 发送停止信号
	close(cm.stopChan)
}

// GetWatchedFiles 获取监控的文件列表
func (cm *ConfigMonitor) GetWatchedFiles() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var files []string
	for path := range cm.watchedFiles {
		files = append(files, path)
	}

	return files
}

// GetReloadCount 获取重载次数
func (cm *ConfigMonitor) GetReloadCount() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.reloadCount
}

// IsWatching 检查是否正在监控指定文件
func (cm *ConfigMonitor) IsWatching(configPath string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	_, exists := cm.watchedFiles[configPath]
	return exists
}

// GetFileInfo 获取文件监控信息
func (cm *ConfigMonitor) GetFileInfo(configPath string) (*FileWatcher, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	watcher, exists := cm.watchedFiles[configPath]
	if !exists {
		return nil, fmt.Errorf("file %s is not being watched", configPath)
	}

	return watcher, nil
}

// RefreshFile 手动刷新文件监控
func (cm *ConfigMonitor) RefreshFile(configPath string) error {
	cm.mu.RLock()
	watcher, exists := cm.watchedFiles[configPath]
	cm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("file %s is not being watched", configPath)
	}

	// 手动检查文件变化
	return cm.checkFileChanges(watcher)
}

// SetReloadInterval 设置重载间隔
func (cm *ConfigMonitor) SetReloadInterval(interval time.Duration) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.reloadInterval = interval

	// 重新启动所有监控器以应用新的间隔
	for _, watcher := range cm.watchedFiles {
		// 停止旧的监控
		close(watcher.stopChan)

		// 创建新的停止信号通道
		watcher.stopChan = make(chan struct{})

		// 重新启动监控
		go cm.watchFile(watcher)
	}
}
