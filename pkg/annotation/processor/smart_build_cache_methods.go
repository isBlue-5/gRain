// Package processor 提供智能增量构建系统的辅助方法
package processor

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/isBlue-5/grain/pkg/annotation/types"
)

// 依赖图相关方法

// NewDependencyGraph 创建新的依赖关系图
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes:       make(map[string]*DependencyNode),
		reverseDeps: make(map[string][]string),
	}
}

// AddDependency 添加依赖关系
func (dg *DependencyGraph) AddDependency(source, target string) {
	dg.mu.Lock()
	defer dg.mu.Unlock()

	// 确保源节点存在
	if dg.nodes[source] == nil {
		dg.nodes[source] = &DependencyNode{
			FilePath:     source,
			DirectDeps:   make([]string, 0),
			IndirectDeps: make([]string, 0),
			LastUpdated:  time.Now(),
		}
	}

	// 添加直接依赖
	node := dg.nodes[source]
	for _, dep := range node.DirectDeps {
		if dep == target {
			return // 依赖已存在
		}
	}
	node.DirectDeps = append(node.DirectDeps, target)
	node.LastUpdated = time.Now()

	// 更新反向依赖映射
	dg.reverseDeps[target] = append(dg.reverseDeps[target], source)

	// 重新计算依赖深度
	dg.calculateDepths()
}

// GetDependents 获取依赖于指定文件的所有文件
func (dg *DependencyGraph) GetDependents(filePath string) []string {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	dependents := make([]string, 0)
	visited := make(map[string]bool)

	dg.collectDependents(filePath, &dependents, visited)

	return dependents
}

// collectDependents 递归收集所有依赖者
func (dg *DependencyGraph) collectDependents(filePath string, dependents *[]string, visited map[string]bool) {
	if visited[filePath] {
		return
	}
	visited[filePath] = true

	// 获取直接依赖者
	if directDeps, exists := dg.reverseDeps[filePath]; exists {
		for _, dep := range directDeps {
			*dependents = append(*dependents, dep)
			dg.collectDependents(dep, dependents, visited)
		}
	}
}

// calculateDepths 计算依赖深度
func (dg *DependencyGraph) calculateDepths() {
	// 拓扑排序计算深度
	inDegree := make(map[string]int)

	// 初始化入度
	for filePath := range dg.nodes {
		inDegree[filePath] = 0
	}

	// 计算入度
	for _, node := range dg.nodes {
		for _, dep := range node.DirectDeps {
			inDegree[dep]++
		}
	}

	// 使用队列进行拓扑排序
	queue := make([]string, 0)
	depths := make(map[string]int)

	// 找到入度为0的节点
	for filePath, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, filePath)
			depths[filePath] = 0
		}
	}

	// 处理队列
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if node, exists := dg.nodes[current]; exists {
			for _, dep := range node.DirectDeps {
				inDegree[dep]--

				// 更新深度
				if depths[dep] < depths[current]+1 {
					depths[dep] = depths[current] + 1
				}

				// 如果入度为0，加入队列
				if inDegree[dep] == 0 {
					queue = append(queue, dep)
				}
			}
		}
	}

	// 更新节点深度
	for filePath, depth := range depths {
		if node, exists := dg.nodes[filePath]; exists {
			node.Depth = depth
		}
	}
}

// SmartBuildCache 的辅助方法

// collectDependencies 收集文件依赖
func (c *SmartBuildCache) collectDependencies(sourceFile string, context *GenerationContext) []string {
	dependencies := make([]string, 0)

	// 收集导入的包依赖
	if imports := c.extractImports(sourceFile); len(imports) > 0 {
		dependencies = append(dependencies, imports...)
	}

	// 收集模板依赖
	for _, templatePath := range context.Templates {
		dependencies = append(dependencies, templatePath)
	}

	// 收集配置文件依赖（如果配置来自文件）
	if configFiles := c.extractConfigFiles(context.Config); len(configFiles) > 0 {
		dependencies = append(dependencies, configFiles...)
	}

	return dependencies
}

// extractImports 提取文件的导入依赖
func (c *SmartBuildCache) extractImports(sourceFile string) []string {
	// 这里简化实现，实际应该解析AST获取导入信息
	// 为了演示，返回空列表
	return []string{}
}

// extractConfigFiles 提取配置文件依赖
func (c *SmartBuildCache) extractConfigFiles(config interface{}) []string {
	// 这里简化实现，实际应该分析配置对象中的文件引用
	return []string{}
}

// updateDependencyGraph 更新依赖关系图
func (c *SmartBuildCache) updateDependencyGraph(sourceFile string, dependencies []string) {
	for _, dep := range dependencies {
		c.depGraph.AddDependency(sourceFile, dep)
	}
}

// isCacheExpired 检查缓存是否过期
func (c *SmartBuildCache) isCacheExpired(entry *BuildCacheEntry) bool {
	if c.options.TTL == 0 {
		return false // 永不过期
	}

	return time.Since(entry.CreatedAt) > c.options.TTL
}

// validateGeneratedFiles 验证生成的文件是否存在且未被修改
func (c *SmartBuildCache) validateGeneratedFiles(entry *BuildCacheEntry) bool {
	for _, fileInfo := range entry.GeneratedFiles {
		// 检查文件是否存在
		if _, err := os.Stat(fileInfo.Path); os.IsNotExist(err) {
			return false
		}

		// 检查文件哈希是否匹配
		currentHash, err := c.calculateFileHash(fileInfo.Path)
		if err != nil || currentHash != fileInfo.Hash {
			return false
		}
	}

	return true
}

// hasDependencyChanged 检查依赖是否发生变化
func (c *SmartBuildCache) hasDependencyChanged(entry *BuildCacheEntry, context *GenerationContext) bool {
	c.stats.DependencyChecks++

	// 检查直接依赖
	for _, dep := range entry.Dependencies {
		if c.isFileChanged(dep, entry.Key.DependencyHashes[dep]) {
			return true
		}
	}

	// 检查间接依赖（通过依赖图）
	if c.options.EnableDependencyTracking {
		for _, dep := range entry.Dependencies {
			if dependents := c.depGraph.GetDependents(dep); len(dependents) > 0 {
				for _, dependent := range dependents {
					if c.isFileChanged(dependent, "") {
						return true
					}
				}
			}
		}
	}

	return false
}

// isFileChanged 检查单个文件是否发生变化
func (c *SmartBuildCache) isFileChanged(filePath, expectedHash string) bool {
	currentHash, err := c.calculateFileHash(filePath)
	if err != nil {
		return true // 无法计算哈希，认为已变化
	}

	return currentHash != expectedHash
}

// getFileInfo 获取文件信息
func (c *SmartBuildCache) getFileInfo(filePath string) (*GeneratedFileInfo, error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	hash, err := c.calculateFileHash(filePath)
	if err != nil {
		return nil, err
	}

	return &GeneratedFileInfo{
		Path:        filePath,
		Hash:        hash,
		Size:        stat.Size(),
		GeneratedAt: time.Now(),
	}, nil
}

// buildAnnotationSummary 构建注解摘要
func (c *SmartBuildCache) buildAnnotationSummary(annotations []types.Annotation) (*AnnotationSummary, error) {
	counts := make(map[string]int)
	var allAttributes []string
	var allSignatures []string

	for _, annotation := range annotations {
		// 统计注解类型
		annotationType := annotation.GetType()
		counts[annotationType]++

		// 收集属性信息
		for _, attr := range annotation.GetAttributes() {
			attrStr := fmt.Sprintf("%s=%v", attr.Name, attr.Value)
			allAttributes = append(allAttributes, attrStr)
		}

		// 收集方法签名信息
		if methodAnno, ok := annotation.(*types.MethodAnnotation); ok {
			signature := fmt.Sprintf("%s.%s", methodAnno.ReceiverType, methodAnno.MethodName)
			allSignatures = append(allSignatures, signature)
		}
	}

	// 计算属性哈希
	attributesData := strings.Join(allAttributes, ";")
	attributesHash := c.calculateStringHash(attributesData)

	// 计算方法签名哈希
	signaturesData := strings.Join(allSignatures, ";")
	signaturesHash := c.calculateStringHash(signaturesData)

	return &AnnotationSummary{
		AnnotationCounts:    counts,
		AttributesHash:      attributesHash,
		MethodSignatureHash: signaturesHash,
	}, nil
}

// calculateStringHash 计算字符串哈希
func (c *SmartBuildCache) calculateStringHash(data string) string {
	hasher := sha256.New()
	hasher.Write([]byte(data))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

// startCleanup 启动清理协程
func (c *SmartBuildCache) startCleanup() {
	ticker := time.NewTicker(c.options.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup 清理过期缓存
func (c *SmartBuildCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	expiredKeys := make([]string, 0)

	// 找到过期的缓存条目
	for key, entry := range c.entries {
		if c.options.TTL > 0 && now.Sub(entry.CreatedAt) > c.options.TTL {
			expiredKeys = append(expiredKeys, key)
		}
	}

	// 删除过期条目
	for _, key := range expiredKeys {
		delete(c.entries, key)
		c.stats.CacheInvalidations++
	}

	// 如果缓存条目过多，删除最少使用的条目
	if c.options.MaxEntries > 0 && len(c.entries) > c.options.MaxEntries {
		c.evictLRU()
	}

	// 更新命中率
	if c.stats.TotalBuildRequests > 0 {
		c.stats.HitRate = float64(c.stats.CacheHits) / float64(c.stats.TotalBuildRequests)
	}

	if c.options.VerboseLogging {
		fmt.Printf("缓存清理完成: 删除了 %d 个过期条目, 当前缓存条目数: %d, 命中率: %.2f%%\n",
			len(expiredKeys), len(c.entries), c.stats.HitRate*100)
	}
}

// evictLRU 使用LRU策略淘汰缓存
func (c *SmartBuildCache) evictLRU() {
	// 按最后访问时间排序，删除最老的条目
	type entryWithKey struct {
		key   string
		entry *BuildCacheEntry
	}

	entries := make([]entryWithKey, 0, len(c.entries))
	for key, entry := range c.entries {
		entries = append(entries, entryWithKey{key, entry})
	}

	// 按最后访问时间排序
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].entry.LastAccessedAt.After(entries[j].entry.LastAccessedAt) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// 删除最老的条目，直到满足数量限制
	toDelete := len(c.entries) - c.options.MaxEntries
	for i := 0; i < toDelete && i < len(entries); i++ {
		delete(c.entries, entries[i].key)
		c.stats.CacheInvalidations++
	}
}

// loadCache 加载持久化的缓存
func (c *SmartBuildCache) loadCache() {
	cachePath := filepath.Join(c.cacheDir, "smart_build_cache.json")

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if c.options.VerboseLogging {
			fmt.Printf("无法加载缓存文件: %v\n", err)
		}
		return
	}

	var persistedCache struct {
		Entries          map[string]*BuildCacheEntry `json:"entries"`
		GeneratorVersion string                      `json:"generator_version"`
		Stats            *BuildCacheStats            `json:"stats"`
	}

	if err := json.Unmarshal(data, &persistedCache); err != nil {
		if c.options.VerboseLogging {
			fmt.Printf("无法解析缓存文件: %v\n", err)
		}
		return
	}

	// 如果生成器版本不匹配，清空缓存
	if persistedCache.GeneratorVersion != c.generatorVersion {
		if c.options.VerboseLogging {
			fmt.Printf("生成器版本不匹配，清空缓存: %s != %s\n",
				persistedCache.GeneratorVersion, c.generatorVersion)
		}
		return
	}

	c.entries = persistedCache.Entries
	if persistedCache.Stats != nil {
		c.stats = persistedCache.Stats
	}

	if c.options.VerboseLogging {
		fmt.Printf("成功加载 %d 个缓存条目\n", len(c.entries))
	}
}

// saveCache 保存缓存到磁盘
func (c *SmartBuildCache) saveCache() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cachePath := filepath.Join(c.cacheDir, "smart_build_cache.json")

	persistedCache := struct {
		Entries          map[string]*BuildCacheEntry `json:"entries"`
		GeneratorVersion string                      `json:"generator_version"`
		Stats            *BuildCacheStats            `json:"stats"`
	}{
		Entries:          c.entries,
		GeneratorVersion: c.generatorVersion,
		Stats:            c.stats,
	}

	data, err := json.MarshalIndent(persistedCache, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化缓存失败: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("保存缓存文件失败: %w", err)
	}

	return nil
}

// GetStats 获取缓存统计信息
func (c *SmartBuildCache) GetStats() *BuildCacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 创建副本以避免并发修改
	stats := *c.stats

	// 更新命中率
	if stats.TotalBuildRequests > 0 {
		stats.HitRate = float64(stats.CacheHits) / float64(stats.TotalBuildRequests)
	}

	return &stats
}

// InvalidateByPattern 按模式失效缓存
func (c *SmartBuildCache) InvalidateByPattern(pattern string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	invalidated := 0
	toDelete := make([]string, 0)

	for key, entry := range c.entries {
		// 检查是否匹配模式
		if c.matchesPattern(entry, pattern) {
			toDelete = append(toDelete, key)
			invalidated++
		}
	}

	// 删除匹配的条目
	for _, key := range toDelete {
		delete(c.entries, key)
	}

	c.stats.CacheInvalidations += int64(invalidated)

	return invalidated
}

// matchesPattern 检查缓存条目是否匹配指定模式
func (c *SmartBuildCache) matchesPattern(entry *BuildCacheEntry, pattern string) bool {
	// 简化实现：检查生成的文件路径是否包含模式
	for _, fileInfo := range entry.GeneratedFiles {
		if strings.Contains(fileInfo.Path, pattern) {
			return true
		}
	}

	// 检查依赖文件路径
	for _, dep := range entry.Dependencies {
		if strings.Contains(dep, pattern) {
			return true
		}
	}

	return false
}

// Close 关闭缓存并保存到磁盘
func (c *SmartBuildCache) Close() error {
	return c.saveCache()
}
