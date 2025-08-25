package processor

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/isBlue-5/grain/pkg/annotation/registry"
	"github.com/isBlue-5/grain/pkg/annotation/types"
)

// Generator 代码生成器主类
type Generator struct {
	// 注解解析器
	parser AnnotationParser

	// 注解注册中心
	registry registry.Registry

	// 依赖注入处理器
	injectProcessor *InjectProcessor

	// 路由注册处理器
	routeProcessor *RouteProcessor

	// 绑定处理器
	bindingProcessor *BindingProcessor

	// 参数绑定处理器
	paramBindingProcessor *ParamBindingProcessor

	// 增强路由处理器
	enhancedRouteProcessor *EnhancedRouteProcessor

	// 日志处理器
	logProcessor *LogProcessor

	// 事务处理器
	transactionProcessor *TransactionProcessor

	// 实体处理器
	entityProcessor *EntityProcessor

	// 服务处理器
	serviceProcessor *ServiceProcessor

	// 仓库处理器
	repositoryProcessor *RepositoryProcessor

	// 缓存处理器
	cacheProcessor *CacheProcessor

	// 查询处理器
	queryProcessor *QueryProcessor

	// 关联关系处理器
	associationProcessor *AssociationProcessor

	// 限流处理器
	rateLimitProcessor *RateLimitProcessor

	// 权限控制处理器
	authzProcessor *AuthzProcessor

	// 输出目录
	outputDir string

	// 是否启用详细日志
	verbose bool

	// 缓存目录
	cacheDir string

	// 增量生成标志
	incremental bool

	// 文件哈希缓存
	fileHashes map[string]string

	// 修改时间缓存
	fileMTimes map[string]time.Time

	// 缓存锁
	cacheMutex sync.RWMutex

	// 统计信息
	stats *GeneratorStats

	// 注解前缀
	prefix string
}

// GeneratorStats 生成器统计信息
type GeneratorStats struct {
	TotalFiles       int                          // 处理的总文件数
	ChangedFiles     int                          // 变更的文件数
	CachedFiles      int                          // 使用缓存的文件数
	ProcessingTime   time.Duration                // 处理时间
	AnnotationCounts map[types.AnnotationType]int // 注解计数
}

// GeneratorOption 生成器选项
type GeneratorOption func(*Generator)

// WithOutputDir 设置输出目录
func WithOutputDir(dir string) GeneratorOption {
	return func(g *Generator) {
		g.outputDir = dir
	}
}

// WithVerbose 设置是否启用详细日志
func WithVerbose(verbose bool) GeneratorOption {
	return func(g *Generator) {
		g.verbose = verbose
	}
}

// WithCacheDir 设置缓存目录
func WithCacheDir(dir string) GeneratorOption {
	return func(g *Generator) {
		g.cacheDir = dir
	}
}

// WithIncremental 设置是否使用增量生成
func WithIncremental(incremental bool) GeneratorOption {
	return func(g *Generator) {
		g.incremental = incremental
	}
}

// NewGenerator 创建代码生成器
func NewGenerator(parser AnnotationParser, reg registry.Registry, outputDir string, prefix string) *Generator {
	// 创建生成器
	g := &Generator{
		parser:      parser,
		registry:    reg,
		outputDir:   outputDir,
		verbose:     false,
		cacheDir:    filepath.Join(outputDir, ".cache"),
		incremental: false,
		fileHashes:  make(map[string]string),
		fileMTimes:  make(map[string]time.Time),
		stats: &GeneratorStats{
			AnnotationCounts: make(map[types.AnnotationType]int),
		},
		prefix: prefix,
	}

	// 创建处理器
	g.injectProcessor = NewInjectProcessor(g.registry, prefix, filepath.Join(g.outputDir, "inject"))
	g.routeProcessor = NewRouteProcessor(g.registry, prefix, filepath.Join(g.outputDir, "route"))
	g.bindingProcessor = NewBindingProcessor(g.registry, prefix, filepath.Join(g.outputDir, "binding"))
	g.paramBindingProcessor = NewParamBindingProcessor(g.registry, g.parser, filepath.Join(g.outputDir, "param_binding"))
	g.enhancedRouteProcessor = NewEnhancedRouteProcessor(g.registry, g.parser, filepath.Join(g.outputDir, "enhanced_route"))
	g.logProcessor = NewLogProcessor(g.registry, prefix, filepath.Join(g.outputDir, "log"))
	// 创建事务处理器
	g.transactionProcessor = NewTransactionProcessor(g.parser, g, filepath.Join(g.outputDir, "transaction"))
	// 创建实体处理器
	g.entityProcessor = NewEntityProcessor(g.registry, g.parser, filepath.Join(g.outputDir, "entity"))
	// 创建服务处理器
	g.serviceProcessor = NewServiceProcessor(g.registry, g.parser, filepath.Join(g.outputDir, "service"))
	// 创建仓库处理器
	g.repositoryProcessor = NewRepositoryProcessor(g.registry, g.parser, filepath.Join(g.outputDir, "repository"))
	// 创建缓存处理器
	g.cacheProcessor = NewCacheProcessor(g.registry, g.parser, filepath.Join(g.outputDir, "cache"))
	// 创建查询处理器
	g.queryProcessor = NewQueryProcessor(g.registry, g.parser, filepath.Join(g.outputDir, "query"))
	// 创建关联关系处理器
	g.associationProcessor = NewAssociationProcessor(g.registry, g.parser, filepath.Join(g.outputDir, "association"))

	// 创建限流处理器
	g.rateLimitProcessor = NewRateLimitProcessor(g.registry, g.parser, filepath.Join(g.outputDir, "ratelimit"))

	// 创建权限控制处理器
	g.authzProcessor = NewAuthzProcessor(g.registry, g.parser, filepath.Join(g.outputDir, "authz"))

	return g
}

// Generate 生成代码
func (g *Generator) Generate(paths []string) error {
	// 记录开始时间
	startTime := time.Now()

	// 解析注解
	if err := g.parseAnnotations(paths); err != nil {
		return err
	}

	// 处理依赖注入
	if err := g.injectProcessor.ProcessInject(paths); err != nil {
		return err
	}

	// 处理路由注册
	if err := g.routeProcessor.ProcessRoute(paths); err != nil {
		return err
	}

	// 处理请求绑定
	if err := g.bindingProcessor.ProcessBinding(paths); err != nil {
		return err
	}

	// 处理参数绑定
	if err := g.paramBindingProcessor.WithVerbose(true).ProcessParamBinding(paths); err != nil {
		return err
	}

	// 处理增强路由
	if err := g.enhancedRouteProcessor.ProcessEnhancedRoutes(paths); err != nil {
		return err
	}

	// 处理日志注解
	if err := g.logProcessor.ProcessLog(paths); err != nil {
		return err
	}

	// 处理事务注解
	if err := g.transactionProcessor.ProcessTransaction(paths); err != nil {
		return err
	}

	// 处理实体注解
	fmt.Printf("DEBUG: 开始处理实体注解...\n")
	if err := g.entityProcessor.ProcessEntity(paths); err != nil {
		fmt.Printf("DEBUG: 实体注解处理失败: %v\n", err)
		return err
	}
	fmt.Printf("DEBUG: 实体注解处理完成\n")

	// 处理服务注解
	fmt.Printf("DEBUG: 开始处理服务注解...\n")
	if err := g.serviceProcessor.ProcessService(paths); err != nil {
		fmt.Printf("DEBUG: 服务注解处理失败: %v\n", err)
		return err
	}
	fmt.Printf("DEBUG: 服务注解处理完成\n")

	// 处理仓库注解
	fmt.Printf("DEBUG: 开始处理仓库注解...\n")
	if err := g.repositoryProcessor.ProcessRepository(paths); err != nil {
		fmt.Printf("DEBUG: 仓库注解处理失败: %v\n", err)
		return err
	}
	fmt.Printf("DEBUG: 仓库注解处理完成\n")

	// 处理缓存注解
	if err := g.cacheProcessor.ProcessCache(paths); err != nil {
		return err
	}

	// 处理查询注解
	if err := g.queryProcessor.ProcessQuery(paths); err != nil {
		return err
	}

	// 处理关联关系注解
	if err := g.associationProcessor.ProcessAssociation(paths); err != nil {
		return err
	}

	// 处理限流注解
	if err := g.rateLimitProcessor.ProcessRateLimit(paths); err != nil {
		return err
	}

	// 处理权限控制注解
	if err := g.authzProcessor.ProcessAuthz(paths); err != nil {
		return err
	}

	// 统计处理时间
	g.stats.ProcessingTime = time.Since(startTime)

	// 验证生成的代码质量
	if err := g.validateGeneratedCode(); err != nil {
		if g.verbose {
			fmt.Printf("代码质量验证失败: %v\n", err)
		}
		// 不返回错误，只记录警告
	}

	// 打印统计信息
	if g.verbose {
		g.printStats()
	}

	return nil
}

// parseAnnotations 解析所有注解
func (g *Generator) parseAnnotations(paths []string) error {
	var allAnnotations []types.Annotation
	var changedPaths []string

	// 处理每个路径
	for _, path := range paths {
		// 如果启用增量生成，检查文件是否变更
		if g.incremental && !g.isPathChanged(path) {
			if g.verbose {
				fmt.Printf("路径 %s 未变更，跳过处理\n", path)
			}
			g.stats.CachedFiles++
			continue
		}

		changedPaths = append(changedPaths, path)

		// 判断路径类型（文件、包或目录）
		annotations, err := g.parser.ParsePackage(path)
		if err != nil {
			// 尝试作为单个文件解析
			annotations, err = g.parser.ParseFile(path)
			if err != nil {
				return fmt.Errorf("解析路径 %s 失败: %w", path, err)
			}
		}

		allAnnotations = append(allAnnotations, annotations...)
		g.stats.ChangedFiles++

		if g.verbose {
			fmt.Printf("从路径 %s 解析到 %d 个注解\n", path, len(annotations))
		}

		// 如果启用增量生成，更新文件哈希
		if g.incremental {
			g.updatePathHash(path)
		}
	}

	g.stats.TotalFiles = g.stats.ChangedFiles + g.stats.CachedFiles

	// 注册所有注解
	g.registry.RegisterAll(allAnnotations)

	// 更新注解统计
	for _, anno := range allAnnotations {
		g.stats.AnnotationCounts[anno.GetType()]++
	}

	if g.verbose {
		fmt.Printf("总共注册了 %d 个注解\n", len(allAnnotations))
	}

	return nil
}

// isPathChanged 检查路径是否变更
func (g *Generator) isPathChanged(path string) bool {
	// 如果是目录，检查目录下所有Go文件
	isDir := false
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		isDir = true
		return g.isDirChanged(path)
	}

	// 单个文件处理
	if !isDir {
		g.cacheMutex.RLock()
		oldHash, exists := g.fileHashes[path]
		g.cacheMutex.RUnlock()

		if !exists {
			return true
		}

		newHash, err := g.calculateFileHash(path)
		if err != nil || newHash != oldHash {
			return true
		}
	}

	return false
}

// isDirChanged 检查目录是否变更
func (g *Generator) isDirChanged(dirPath string) bool {
	changed := false

	// 遍历目录下所有Go文件
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 忽略目录和非Go文件
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		g.cacheMutex.RLock()
		oldHash, exists := g.fileHashes[path]
		oldMTime, timeExists := g.fileMTimes[path]
		g.cacheMutex.RUnlock()

		// 首先检查修改时间是否变化（快速检查）
		if timeExists && !info.ModTime().After(oldMTime) {
			return nil
		}

		// 如果修改时间变化或不存在哈希，计算哈希
		newHash, err := g.calculateFileHash(path)
		if err != nil || !exists || newHash != oldHash {
			changed = true
			return filepath.SkipDir // 一旦发现变化，停止遍历
		}

		return nil
	})

	if err != nil {
		// 如果遍历失败，保守地认为目录已变更
		return true
	}

	return changed
}

// updatePathHash 更新路径的哈希值
func (g *Generator) updatePathHash(path string) {
	// 如果是目录，更新目录下所有Go文件
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		g.updateDirHash(path)
		return
	}

	// 单个文件处理
	hash, err := g.calculateFileHash(path)
	if err != nil {
		return
	}

	g.cacheMutex.Lock()
	defer g.cacheMutex.Unlock()

	g.fileHashes[path] = hash
	if info, err := os.Stat(path); err == nil {
		g.fileMTimes[path] = info.ModTime()
	}
}

// updateDirHash 更新目录的哈希值
func (g *Generator) updateDirHash(dirPath string) {
	// 遍历目录下所有Go文件
	filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// 忽略目录和非Go文件
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		hash, err := g.calculateFileHash(path)
		if err != nil {
			return nil
		}

		g.cacheMutex.Lock()
		g.fileHashes[path] = hash
		g.fileMTimes[path] = info.ModTime()
		g.cacheMutex.Unlock()

		return nil
	})
}

// calculateFileHash 计算文件哈希值
func (g *Generator) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// loadCache 加载缓存
func (g *Generator) loadCache() {
	// 确保缓存目录存在
	if err := os.MkdirAll(g.cacheDir, 0755); err != nil {
		if g.verbose {
			fmt.Printf("创建缓存目录失败: %v\n", err)
		}
		return
	}

	// 加载文件哈希缓存
	hashesPath := filepath.Join(g.cacheDir, "file_hashes.txt")
	hashesData, err := os.ReadFile(hashesPath)
	if err == nil {
		lines := strings.Split(string(hashesData), "\n")
		for _, line := range lines {
			parts := strings.SplitN(line, " ", 2)
			if len(parts) == 2 {
				g.fileHashes[parts[1]] = parts[0]
			}
		}

		if g.verbose {
			fmt.Printf("从缓存加载了 %d 条文件哈希记录\n", len(g.fileHashes))
		}
	}

	// 加载文件修改时间缓存
	mtimesPath := filepath.Join(g.cacheDir, "file_mtimes.txt")
	mtimesData, err := os.ReadFile(mtimesPath)
	if err == nil {
		lines := strings.Split(string(mtimesData), "\n")
		for _, line := range lines {
			parts := strings.SplitN(line, " ", 2)
			if len(parts) == 2 {
				timestamp, err := time.Parse(time.RFC3339, parts[0])
				if err == nil {
					g.fileMTimes[parts[1]] = timestamp
				}
			}
		}

		if g.verbose {
			fmt.Printf("从缓存加载了 %d 条文件修改时间记录\n", len(g.fileMTimes))
		}
	}
}

// saveCache 保存缓存
func (g *Generator) saveCache() {
	// 确保缓存目录存在
	if err := os.MkdirAll(g.cacheDir, 0755); err != nil {
		if g.verbose {
			fmt.Printf("创建缓存目录失败: %v\n", err)
		}
		return
	}

	// 保存文件哈希缓存
	var hashesBuilder strings.Builder
	g.cacheMutex.RLock()
	for path, hash := range g.fileHashes {
		hashesBuilder.WriteString(hash)
		hashesBuilder.WriteString(" ")
		hashesBuilder.WriteString(path)
		hashesBuilder.WriteString("\n")
	}
	g.cacheMutex.RUnlock()

	hashesPath := filepath.Join(g.cacheDir, "file_hashes.txt")
	if err := os.WriteFile(hashesPath, []byte(hashesBuilder.String()), 0644); err != nil && g.verbose {
		fmt.Printf("保存文件哈希缓存失败: %v\n", err)
	}

	// 保存文件修改时间缓存
	var mtimesBuilder strings.Builder
	g.cacheMutex.RLock()
	for path, mtime := range g.fileMTimes {
		mtimesBuilder.WriteString(mtime.Format(time.RFC3339))
		mtimesBuilder.WriteString(" ")
		mtimesBuilder.WriteString(path)
		mtimesBuilder.WriteString("\n")
	}
	g.cacheMutex.RUnlock()

	mtimesPath := filepath.Join(g.cacheDir, "file_mtimes.txt")
	if err := os.WriteFile(mtimesPath, []byte(mtimesBuilder.String()), 0644); err != nil && g.verbose {
		fmt.Printf("保存文件修改时间缓存失败: %v\n", err)
	}
}

// CountAnnotationsByType 按类型统计注解数量
func (g *Generator) CountAnnotationsByType() map[types.AnnotationType]int {
	return g.stats.AnnotationCounts
}

// GetRegistry 获取注册中心
func (g *Generator) GetRegistry() registry.Registry {
	return g.registry
}

// GetOutputDir 获取输出目录
func (g *Generator) GetOutputDir() string {
	return g.outputDir
}

// GetStats 获取统计信息
func (g *Generator) GetStats() *GeneratorStats {
	return g.stats
}

// validateGeneratedCode 验证生成的代码质量
func (g *Generator) validateGeneratedCode() error {
	// 使用代码验证器验证生成的代码
	return ValidateGeneratedCode(g.outputDir)
}

// printStats 打印统计信息
func (g *Generator) printStats() {
	fmt.Println("\n--- 代码生成统计信息 ---")
	fmt.Printf("处理总文件数: %d\n", g.stats.TotalFiles)
	fmt.Printf("变更文件数: %d\n", g.stats.ChangedFiles)
	fmt.Printf("缓存文件数: %d\n", g.stats.CachedFiles)
	fmt.Printf("处理时间: %v\n", g.stats.ProcessingTime)

	fmt.Println("\n注解统计:")
	for annoType, count := range g.stats.AnnotationCounts {
		fmt.Printf("  - %s: %d 个\n", annoType, count)
	}

	fmt.Println("------------------------")
}
