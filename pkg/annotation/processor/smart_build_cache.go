// Package processor 提供智能增量构建系统
package processor

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/isBlue-5/grain/pkg/annotation/types"
)

// SmartBuildCache 智能构建缓存系统
type SmartBuildCache struct {
	// 缓存条目
	entries map[string]*BuildCacheEntry

	// 依赖关系图
	depGraph *DependencyGraph

	// 模板版本映射
	templateVersions map[string]string

	// 配置哈希
	configHash string

	// 生成器版本
	generatorVersion string

	// 缓存目录
	cacheDir string

	// 并发锁
	mu sync.RWMutex

	// 统计信息
	stats *BuildCacheStats

	// 配置选项
	options *SmartCacheOptions
}

// BuildCacheEntry 构建缓存条目
type BuildCacheEntry struct {
	// 多维度缓存键
	Key *MultiDimensionCacheKey `json:"key"`

	// 生成的文件信息
	GeneratedFiles []GeneratedFileInfo `json:"generated_files"`

	// 缓存时间
	CreatedAt time.Time `json:"created_at"`

	// 最后访问时间
	LastAccessedAt time.Time `json:"last_accessed_at"`

	// 访问次数
	AccessCount int64 `json:"access_count"`

	// 依赖文件列表
	Dependencies []string `json:"dependencies"`

	// 注解信息摘要
	AnnotationSummary *AnnotationSummary `json:"annotation_summary"`
}

// MultiDimensionCacheKey 多维度缓存键
type MultiDimensionCacheKey struct {
	// 源文件哈希
	SourceFileHash string `json:"source_file_hash"`

	// 生成器版本哈希
	GeneratorVersion string `json:"generator_version"`

	// 模板文件哈希映射
	TemplateHashes map[string]string `json:"template_hashes"`

	// 配置哈希
	ConfigHash string `json:"config_hash"`

	// 依赖文件哈希
	DependencyHashes map[string]string `json:"dependency_hashes"`

	// AST结构哈希（注解相关部分）
	ASTStructureHash string `json:"ast_structure_hash"`

	// 编译器版本
	CompilerVersion string `json:"compiler_version"`

	// 环境变量哈希（影响生成的环境变量）
	EnvHash string `json:"env_hash"`
}

// GeneratedFileInfo 生成文件信息
type GeneratedFileInfo struct {
	// 文件路径
	Path string `json:"path"`

	// 文件哈希
	Hash string `json:"hash"`

	// 文件大小
	Size int64 `json:"size"`

	// 生成时间
	GeneratedAt time.Time `json:"generated_at"`
}

// AnnotationSummary 注解信息摘要
type AnnotationSummary struct {
	// 注解数量按类型统计
	AnnotationCounts map[string]int `json:"annotation_counts"`

	// 注解属性哈希
	AttributesHash string `json:"attributes_hash"`

	// 方法签名哈希
	MethodSignatureHash string `json:"method_signature_hash"`
}

// DependencyGraph 依赖关系图
type DependencyGraph struct {
	// 节点映射 (文件路径 -> 依赖节点)
	nodes map[string]*DependencyNode

	// 反向依赖映射 (被依赖文件 -> 依赖它的文件列表)
	reverseDeps map[string][]string

	// 锁
	mu sync.RWMutex
}

// DependencyNode 依赖节点
type DependencyNode struct {
	// 文件路径
	FilePath string

	// 直接依赖
	DirectDeps []string

	// 间接依赖
	IndirectDeps []string

	// 依赖深度
	Depth int

	// 最后更新时间
	LastUpdated time.Time
}

// BuildCacheStats 构建缓存统计
type BuildCacheStats struct {
	// 缓存命中次数
	CacheHits int64 `json:"cache_hits"`

	// 缓存未命中次数
	CacheMisses int64 `json:"cache_misses"`

	// 缓存失效次数
	CacheInvalidations int64 `json:"cache_invalidations"`

	// 依赖变化检测次数
	DependencyChecks int64 `json:"dependency_checks"`

	// 节省的构建时间（毫秒）
	TimeSavedMs int64 `json:"time_saved_ms"`

	// 总的构建请求次数
	TotalBuildRequests int64 `json:"total_build_requests"`

	// 缓存命中率
	HitRate float64 `json:"hit_rate"`
}

// SmartCacheOptions 智能缓存选项
type SmartCacheOptions struct {
	// 最大缓存条目数
	MaxEntries int

	// 缓存过期时间
	TTL time.Duration

	// 是否启用依赖跟踪
	EnableDependencyTracking bool

	// 是否启用AST结构哈希
	EnableASTHashing bool

	// 是否启用并行处理
	EnableParallelProcessing bool

	// 最大并发数
	MaxConcurrency int

	// 缓存清理间隔
	CleanupInterval time.Duration

	// 详细日志
	VerboseLogging bool
}

// NewSmartBuildCache 创建智能构建缓存
func NewSmartBuildCache(cacheDir string, options *SmartCacheOptions) *SmartBuildCache {
	if options == nil {
		options = DefaultSmartCacheOptions()
	}

	cache := &SmartBuildCache{
		entries:          make(map[string]*BuildCacheEntry),
		depGraph:         NewDependencyGraph(),
		templateVersions: make(map[string]string),
		cacheDir:         cacheDir,
		stats:            &BuildCacheStats{},
		options:          options,
	}

	// 确保缓存目录存在
	os.MkdirAll(cacheDir, 0755)

	// 加载现有缓存
	cache.loadCache()

	// 启动清理协程
	if options.CleanupInterval > 0 {
		go cache.startCleanup()
	}

	return cache
}

// DefaultSmartCacheOptions 默认智能缓存选项
func DefaultSmartCacheOptions() *SmartCacheOptions {
	return &SmartCacheOptions{
		MaxEntries:               10000,
		TTL:                      24 * time.Hour,
		EnableDependencyTracking: true,
		EnableASTHashing:         true,
		EnableParallelProcessing: true,
		MaxConcurrency:           4,
		CleanupInterval:          1 * time.Hour,
		VerboseLogging:           false,
	}
}

// ShouldRegenerate 检查是否需要重新生成
func (c *SmartBuildCache) ShouldRegenerate(sourceFile string, context *GenerationContext) (bool, *CacheDecision) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	c.stats.TotalBuildRequests++

	// 构建缓存键
	cacheKey, err := c.buildCacheKey(sourceFile, context)
	if err != nil {
		return true, &CacheDecision{
			ShouldRegenerate: true,
			Reason:           "构建缓存键失败",
			Error:            err,
		}
	}

	// 查找现有缓存条目
	keyStr := cacheKey.String()
	entry, exists := c.entries[keyStr]

	if !exists {
		c.stats.CacheMisses++
		return true, &CacheDecision{
			ShouldRegenerate: true,
			Reason:           "缓存条目不存在",
		}
	}

	// 检查缓存是否过期
	if c.isCacheExpired(entry) {
		c.stats.CacheMisses++
		return true, &CacheDecision{
			ShouldRegenerate: true,
			Reason:           "缓存已过期",
		}
	}

	// 检查生成的文件是否存在且未被修改
	if !c.validateGeneratedFiles(entry) {
		c.stats.CacheMisses++
		return true, &CacheDecision{
			ShouldRegenerate: true,
			Reason:           "生成的文件不存在或已被修改",
		}
	}

	// 检查依赖是否变化
	if c.options.EnableDependencyTracking {
		if c.hasDependencyChanged(entry, context) {
			c.stats.CacheMisses++
			return true, &CacheDecision{
				ShouldRegenerate: true,
				Reason:           "依赖文件发生变化",
			}
		}
	}

	// 缓存命中
	c.stats.CacheHits++
	entry.LastAccessedAt = time.Now()
	entry.AccessCount++

	return false, &CacheDecision{
		ShouldRegenerate: false,
		Reason:           "缓存命中",
		CacheEntry:       entry,
	}
}

// RecordGeneration 记录生成结果
func (c *SmartBuildCache) RecordGeneration(sourceFile string, context *GenerationContext, generatedFiles []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 构建缓存键
	cacheKey, err := c.buildCacheKey(sourceFile, context)
	if err != nil {
		return fmt.Errorf("构建缓存键失败: %w", err)
	}

	// 收集生成文件信息
	var fileInfos []GeneratedFileInfo
	for _, filePath := range generatedFiles {
		info, err := c.getFileInfo(filePath)
		if err != nil {
			continue
		}
		fileInfos = append(fileInfos, *info)
	}

	// 构建注解摘要
	summary, err := c.buildAnnotationSummary(context.Annotations)
	if err != nil {
		return fmt.Errorf("构建注解摘要失败: %w", err)
	}

	// 收集依赖信息
	dependencies := c.collectDependencies(sourceFile, context)

	// 创建缓存条目
	entry := &BuildCacheEntry{
		Key:               cacheKey,
		GeneratedFiles:    fileInfos,
		CreatedAt:         time.Now(),
		LastAccessedAt:    time.Now(),
		AccessCount:       1,
		Dependencies:      dependencies,
		AnnotationSummary: summary,
	}

	// 存储缓存条目
	keyStr := cacheKey.String()
	c.entries[keyStr] = entry

	// 更新依赖图
	if c.options.EnableDependencyTracking {
		c.updateDependencyGraph(sourceFile, dependencies)
	}

	return nil
}

// buildCacheKey 构建多维度缓存键
func (c *SmartBuildCache) buildCacheKey(sourceFile string, context *GenerationContext) (*MultiDimensionCacheKey, error) {
	// 计算源文件哈希
	sourceHash, err := c.calculateFileHash(sourceFile)
	if err != nil {
		return nil, fmt.Errorf("计算源文件哈希失败: %w", err)
	}

	// 计算模板文件哈希
	templateHashes := make(map[string]string)
	for templateName, templatePath := range context.Templates {
		hash, err := c.calculateFileHash(templatePath)
		if err != nil {
			continue
		}
		templateHashes[templateName] = hash
	}

	// 计算依赖文件哈希
	depHashes := make(map[string]string)
	if c.options.EnableDependencyTracking {
		deps := c.collectDependencies(sourceFile, context)
		for _, dep := range deps {
			hash, err := c.calculateFileHash(dep)
			if err != nil {
				continue
			}
			depHashes[dep] = hash
		}
	}

	// 计算AST结构哈希
	var astHash string
	if c.options.EnableASTHashing {
		astHash, err = c.calculateASTStructureHash(sourceFile, context.Annotations)
		if err != nil {
			// AST哈希失败不是致命错误
			astHash = ""
		}
	}

	// 计算配置哈希
	configHash := c.calculateConfigHash(context.Config)

	// 计算环境变量哈希
	envHash := c.calculateEnvHash(context.EnvVars)

	return &MultiDimensionCacheKey{
		SourceFileHash:   sourceHash,
		GeneratorVersion: c.generatorVersion,
		TemplateHashes:   templateHashes,
		ConfigHash:       configHash,
		DependencyHashes: depHashes,
		ASTStructureHash: astHash,
		CompilerVersion:  context.CompilerVersion,
		EnvHash:          envHash,
	}, nil
}

// calculateASTStructureHash 计算AST结构哈希（仅包含注解相关部分）
func (c *SmartBuildCache) calculateASTStructureHash(sourceFile string, annotations []types.Annotation) (string, error) {
	// 解析AST
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, sourceFile, nil, parser.ParseComments)
	if err != nil {
		return "", err
	}

	// 提取注解相关的AST结构
	structureInfo := c.extractAnnotationStructure(node, annotations)

	// 计算结构哈希
	data, err := json.Marshal(structureInfo)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// extractAnnotationStructure 提取注解相关的AST结构信息
func (c *SmartBuildCache) extractAnnotationStructure(node ast.Node, annotations []types.Annotation) map[string]interface{} {
	structure := make(map[string]interface{})

	// 遍历AST，提取方法签名、类型定义等关键结构
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			if x.Name != nil {
				// 提取方法签名信息
				funcInfo := map[string]interface{}{
					"name":     x.Name.Name,
					"params":   c.extractParamTypes(x.Type.Params),
					"results":  c.extractParamTypes(x.Type.Results),
					"receiver": c.extractReceiverType(x.Recv),
				}
				structure[fmt.Sprintf("func_%s", x.Name.Name)] = funcInfo
			}
		case *ast.TypeSpec:
			if x.Name != nil {
				// 提取类型定义信息
				typeInfo := map[string]interface{}{
					"name": x.Name.Name,
					"type": c.extractTypeInfo(x.Type),
				}
				structure[fmt.Sprintf("type_%s", x.Name.Name)] = typeInfo
			}
		}
		return true
	})

	return structure
}

// extractParamTypes 提取参数类型信息
func (c *SmartBuildCache) extractParamTypes(fields *ast.FieldList) []string {
	if fields == nil {
		return nil
	}

	var types []string
	for _, field := range fields.List {
		typeStr := c.extractTypeInfo(field.Type)
		types = append(types, typeStr)
	}

	return types
}

// extractReceiverType 提取接收者类型信息
func (c *SmartBuildCache) extractReceiverType(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}

	return c.extractTypeInfo(recv.List[0].Type)
}

// extractTypeInfo 提取类型信息
func (c *SmartBuildCache) extractTypeInfo(expr ast.Expr) string {
	if expr == nil {
		return ""
	}

	switch x := expr.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.StarExpr:
		return "*" + c.extractTypeInfo(x.X)
	case *ast.SelectorExpr:
		return c.extractTypeInfo(x.X) + "." + x.Sel.Name
	case *ast.ArrayType:
		return "[]" + c.extractTypeInfo(x.Elt)
	case *ast.MapType:
		return "map[" + c.extractTypeInfo(x.Key) + "]" + c.extractTypeInfo(x.Value)
	default:
		return fmt.Sprintf("%T", x)
	}
}

// calculateConfigHash 计算配置哈希
func (c *SmartBuildCache) calculateConfigHash(config interface{}) string {
	if config == nil {
		return ""
	}

	data, err := json.Marshal(config)
	if err != nil {
		return ""
	}

	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

// calculateEnvHash 计算环境变量哈希
func (c *SmartBuildCache) calculateEnvHash(envVars []string) string {
	if len(envVars) == 0 {
		return ""
	}

	// 排序确保一致性
	sorted := make([]string, len(envVars))
	copy(sorted, envVars)
	sort.Strings(sorted)

	// 获取环境变量值
	var values []string
	for _, envVar := range sorted {
		value := os.Getenv(envVar)
		values = append(values, fmt.Sprintf("%s=%s", envVar, value))
	}

	data := strings.Join(values, "\n")
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

// calculateFileHash 计算文件哈希
func (c *SmartBuildCache) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// String 返回缓存键的字符串表示
func (k *MultiDimensionCacheKey) String() string {
	data, _ := json.Marshal(k)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// Equals 比较两个缓存键是否相等
func (k *MultiDimensionCacheKey) Equals(other *MultiDimensionCacheKey) bool {
	return k.String() == other.String()
}

// GenerationContext 生成上下文
type GenerationContext struct {
	// 注解列表
	Annotations []types.Annotation

	// 模板映射
	Templates map[string]string

	// 配置信息
	Config interface{}

	// 环境变量列表
	EnvVars []string

	// 编译器版本
	CompilerVersion string
}

// CacheDecision 缓存决策
type CacheDecision struct {
	// 是否需要重新生成
	ShouldRegenerate bool

	// 决策原因
	Reason string

	// 错误信息
	Error error

	// 缓存条目（如果命中）
	CacheEntry *BuildCacheEntry
}

// 其他辅助方法将在后续实现...
