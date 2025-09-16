package processor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"
)

// TemplateEngine 模板引擎接口
type TemplateEngine interface {
	LoadTemplate(name string, content string) error
	LoadTemplateFile(name string, path string) error
	LoadTemplateDir(dir string) error
	ExecuteTemplate(name string, data interface{}) (string, error)
	RegisterFunction(name string, fn interface{}) error
	ValidateTemplate(name string) error
	GetTemplate(name string) (*Template, error)
	ListTemplates() []string
	ReloadTemplates() error
}

// Template 模板结构
type Template struct {
	Name         string                 `json:"name"`
	Content      string                 `json:"content"`
	Path         string                 `json:"path,omitempty"`
	Parent       string                 `json:"parent,omitempty"`
	Blocks       map[string]string      `json:"blocks,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Dependencies []string               `json:"dependencies,omitempty"`
	LastModified time.Time              `json:"last_modified"`
	Compiled     *template.Template     `json:"-"`
	Validated    bool                   `json:"validated"`
}

// TemplateFunction 模板函数定义
type TemplateFunction struct {
	Name        string      `json:"name"`
	Function    interface{} `json:"-"`
	Description string      `json:"description"`
	Examples    []string    `json:"examples,omitempty"`
}

// TemplateValidator 模板验证器
type TemplateValidator struct {
	rules []ValidationRule
}

// ValidationRule 验证规则
type ValidationRule struct {
	Name        string
	Pattern     *regexp.Regexp
	Severity    ErrorSeverity
	Message     string
	Suggestion  string
}

// ValidationResult 验证结果
type ValidationResult struct {
	IsValid     bool                    `json:"is_valid"`
	Errors      []TemplateError         `json:"errors,omitempty"`
	Warnings    []TemplateError         `json:"warnings,omitempty"`
	Suggestions []string                `json:"suggestions,omitempty"`
	Metadata    map[string]interface{}  `json:"metadata,omitempty"`
}

// TemplateError 模板错误
type TemplateError struct {
	Line        int           `json:"line"`
	Column      int           `json:"column"`
	Code        ErrorCode     `json:"code"`
	Severity    ErrorSeverity `json:"severity"`
	Message     string        `json:"message"`
	Suggestion  string        `json:"suggestion,omitempty"`
}

// ModernTemplateEngine 现代化模板引擎
type ModernTemplateEngine struct {
	templates     map[string]*Template
	functions     map[string]*TemplateFunction
	validator     *TemplateValidator
	baseDir       string
	hotReload     bool
	mu            sync.RWMutex
	funcMap       template.FuncMap
	inheritance   map[string][]string // 模板继承关系
	watchers      map[string]*time.Time // 文件监视器
}

// NewModernTemplateEngine 创建现代化模板引擎
func NewModernTemplateEngine(baseDir string, hotReload bool) *ModernTemplateEngine {
	engine := &ModernTemplateEngine{
		templates:   make(map[string]*Template),
		functions:   make(map[string]*TemplateFunction),
		validator:   NewTemplateValidator(),
		baseDir:     baseDir,
		hotReload:   hotReload,
		funcMap:     make(template.FuncMap),
		inheritance: make(map[string][]string),
		watchers:    make(map[string]*time.Time),
	}
	
	// 注册默认函数
	engine.registerDefaultFunctions()
	
	return engine
}

// registerDefaultFunctions 注册默认模板函数
func (mte *ModernTemplateEngine) registerDefaultFunctions() {
	// 字符串处理函数
	mte.RegisterFunction("upper", strings.ToUpper)
	mte.RegisterFunction("lower", strings.ToLower)
	mte.RegisterFunction("title", strings.Title)
	mte.RegisterFunction("trim", strings.TrimSpace)
	mte.RegisterFunction("replace", strings.ReplaceAll)
	mte.RegisterFunction("contains", strings.Contains)
	mte.RegisterFunction("hasPrefix", strings.HasPrefix)
	mte.RegisterFunction("hasSuffix", strings.HasSuffix)
	
	// 格式化函数
	mte.RegisterFunction("sprintf", fmt.Sprintf)
	mte.RegisterFunction("join", strings.Join)
	mte.RegisterFunction("split", strings.Split)
	
	// 类型检查函数
	mte.RegisterFunction("isString", func(v interface{}) bool {
		_, ok := v.(string)
		return ok
	})
	mte.RegisterFunction("isInt", func(v interface{}) bool {
		_, ok := v.(int)
		return ok
	})
	mte.RegisterFunction("isBool", func(v interface{}) bool {
		_, ok := v.(bool)
		return ok
	})
	
	// Go代码生成辅助函数
	mte.RegisterFunction("goType", mte.goTypeHelper)
	mte.RegisterFunction("goValue", mte.goValueHelper)
	mte.RegisterFunction("goImport", mte.goImportHelper)
	mte.RegisterFunction("goPkg", mte.goPkgHelper)
	mte.RegisterFunction("goFormat", mte.goFormatHelper)
	
	// 模板继承函数
	mte.RegisterFunction("extends", mte.extendsHelper)
	mte.RegisterFunction("block", mte.blockHelper)
	mte.RegisterFunction("super", mte.superHelper)
	
	// 条件和循环辅助函数
	mte.RegisterFunction("default", mte.defaultHelper)
	mte.RegisterFunction("empty", mte.emptyHelper)
	mte.RegisterFunction("len", mte.lenHelper)
	mte.RegisterFunction("range", mte.rangeHelper)
}

// Go代码生成辅助函数实现
func (mte *ModernTemplateEngine) goTypeHelper(v interface{}) string {
	if v == nil {
		return "interface{}"
	}
	
	t := reflect.TypeOf(v)
	return t.String()
}

func (mte *ModernTemplateEngine) goValueHelper(v interface{}) string {
	if v == nil {
		return "nil"
	}
	
	switch val := v.(type) {
	case string:
		return fmt.Sprintf(`"%s"`, val)
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", val)
	case float32, float64:
		return fmt.Sprintf("%f", val)
	case bool:
		return fmt.Sprintf("%t", val)
	default:
		return fmt.Sprintf("%#v", val)
	}
}

func (mte *ModernTemplateEngine) goImportHelper(pkg string) string {
	if strings.Contains(pkg, "/") {
		return fmt.Sprintf(`"%s"`, pkg)
	}
	return pkg
}

func (mte *ModernTemplateEngine) goPkgHelper(fullPkg string) string {
	parts := strings.Split(fullPkg, "/")
	return parts[len(parts)-1]
}

func (mte *ModernTemplateEngine) goFormatHelper(code string) (string, error) {
	formatted, err := format.Source([]byte(code))
	if err != nil {
		return code, err
	}
	return string(formatted), nil
}

// 模板继承辅助函数
func (mte *ModernTemplateEngine) extendsHelper(parent string) string {
	// 记录继承关系
	return fmt.Sprintf("{{/* extends %s */}}", parent)
}

func (mte *ModernTemplateEngine) blockHelper(name string, content string) string {
	return fmt.Sprintf("{{/* block %s */}}%s{{/* endblock */}}", name, content)
}

func (mte *ModernTemplateEngine) superHelper() string {
	return "{{/* super */}}"
}

// 其他辅助函数
func (mte *ModernTemplateEngine) defaultHelper(value, defaultValue interface{}) interface{} {
	if value == nil || (reflect.ValueOf(value).Kind() == reflect.String && value.(string) == "") {
		return defaultValue
	}
	return value
}

func (mte *ModernTemplateEngine) emptyHelper(value interface{}) bool {
	if value == nil {
		return true
	}
	
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String, reflect.Array, reflect.Slice, reflect.Map, reflect.Chan:
		return v.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	default:
		return false
	}
}

func (mte *ModernTemplateEngine) lenHelper(value interface{}) int {
	if value == nil {
		return 0
	}
	
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String, reflect.Array, reflect.Slice, reflect.Map, reflect.Chan:
		return v.Len()
	default:
		return 0
	}
}

func (mte *ModernTemplateEngine) rangeHelper(start, end int) []int {
	result := make([]int, end-start)
	for i := start; i < end; i++ {
		result[i-start] = i
	}
	return result
}

// LoadTemplate 加载模板内容
func (mte *ModernTemplateEngine) LoadTemplate(name string, content string) error {
	mte.mu.Lock()
	defer mte.mu.Unlock()
	
	tmpl := &Template{
		Name:         name,
		Content:      content,
		LastModified: time.Now(),
		Blocks:       make(map[string]string),
		Metadata:     make(map[string]interface{}),
	}
	
	// 解析模板元数据
	if err := mte.parseTemplateMetadata(tmpl); err != nil {
		return fmt.Errorf("failed to parse template metadata: %w", err)
	}
	
	// 编译模板
	if err := mte.compileTemplate(tmpl); err != nil {
		return fmt.Errorf("failed to compile template: %w", err)
	}
	
	mte.templates[name] = tmpl
	return nil
}

// LoadTemplateFile 从文件加载模板
func (mte *ModernTemplateEngine) LoadTemplateFile(name string, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read template file %s: %w", path, err)
	}
	
	err = mte.LoadTemplate(name, string(content))
	if err != nil {
		return err
	}
	
	// 记录文件路径和修改时间
	mte.mu.Lock()
	tmpl := mte.templates[name]
	tmpl.Path = path
	
	if stat, err := os.Stat(path); err == nil {
		modTime := stat.ModTime()
		mte.watchers[path] = &modTime
	}
	mte.mu.Unlock()
	
	return nil
}

// LoadTemplateDir 从目录加载所有模板
func (mte *ModernTemplateEngine) LoadTemplateDir(dir string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if d.IsDir() || !strings.HasSuffix(path, ".tmpl") {
			return nil
		}
		
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		
		name := strings.TrimSuffix(relPath, ".tmpl")
		name = strings.ReplaceAll(name, string(filepath.Separator), "/")
		
		return mte.LoadTemplateFile(name, path)
	})
}

// parseTemplateMetadata 解析模板元数据
func (mte *ModernTemplateEngine) parseTemplateMetadata(tmpl *Template) error {
	content := tmpl.Content
	
	// 解析继承关系
	extendsRegex := regexp.MustCompile(`{{\s*/\*\s*extends\s+(\S+)\s*\*/\s*}}`)
	if matches := extendsRegex.FindStringSubmatch(content); len(matches) > 1 {
		tmpl.Parent = matches[1]
		mte.inheritance[tmpl.Name] = append(mte.inheritance[tmpl.Name], tmpl.Parent)
	}
	
	// 解析块定义
	blockRegex := regexp.MustCompile(`{{\s*/\*\s*block\s+(\S+)\s*\*/\s*}}(.*?){{\s*/\*\s*endblock\s*\*/\s*}}`)
	blockMatches := blockRegex.FindAllStringSubmatch(content, -1)
	for _, match := range blockMatches {
		if len(match) > 2 {
			blockName := match[1]
			blockContent := match[2]
			tmpl.Blocks[blockName] = blockContent
		}
	}
	
	// 解析依赖关系
	includeRegex := regexp.MustCompile(`{{\s*template\s+"([^"]+)"\s*.*?}}`)
	includeMatches := includeRegex.FindAllStringSubmatch(content, -1)
	for _, match := range includeMatches {
		if len(match) > 1 {
			depName := match[1]
			tmpl.Dependencies = append(tmpl.Dependencies, depName)
		}
	}
	
	return nil
}

// compileTemplate 编译模板
func (mte *ModernTemplateEngine) compileTemplate(tmpl *Template) error {
	// 处理模板继承
	content := mte.processInheritance(tmpl)
	
	// 创建并编译模板
	t := template.New(tmpl.Name).Funcs(mte.funcMap)
	
	compiled, err := t.Parse(content)
	if err != nil {
		return fmt.Errorf("template compilation failed: %w", err)
	}
	
	tmpl.Compiled = compiled
	return nil
}

// processInheritance 处理模板继承
func (mte *ModernTemplateEngine) processInheritance(tmpl *Template) string {
	content := tmpl.Content
	
	// 如果没有父模板，直接返回
	if tmpl.Parent == "" {
		return content
	}
	
	// 获取父模板
	mte.mu.RLock()
	parentTmpl, exists := mte.templates[tmpl.Parent]
	mte.mu.RUnlock()
	
	if !exists {
		// 父模板不存在，返回原内容
		return content
	}
	
	// 合并父模板和子模板的块
	mergedContent := parentTmpl.Content
	
	// 替换父模板中的块
	for blockName, blockContent := range tmpl.Blocks {
		blockPattern := fmt.Sprintf(`{{\s*/\*\s*block\s+%s\s*\*/\s*}}.*?{{\s*/\*\s*endblock\s*\*/\s*}}`, regexp.QuoteMeta(blockName))
		blockRegex := regexp.MustCompile(blockPattern)
		replacement := fmt.Sprintf("{{/* block %s */}}%s{{/* endblock */}}", blockName, blockContent)
		mergedContent = blockRegex.ReplaceAllString(mergedContent, replacement)
	}
	
	return mergedContent
}

// ExecuteTemplate 执行模板
func (mte *ModernTemplateEngine) ExecuteTemplate(name string, data interface{}) (string, error) {
	// 热重载检查
	if mte.hotReload {
		if err := mte.checkAndReload(name); err != nil {
			return "", fmt.Errorf("hot reload failed: %w", err)
		}
	}
	
	mte.mu.RLock()
	tmpl, exists := mte.templates[name]
	mte.mu.RUnlock()
	
	if !exists {
		return "", fmt.Errorf("template %s not found", name)
	}
	
	if tmpl.Compiled == nil {
		return "", fmt.Errorf("template %s not compiled", name)
	}
	
	var buf bytes.Buffer
	if err := tmpl.Compiled.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("template execution failed: %w", err)
	}
	
	return buf.String(), nil
}

// checkAndReload 检查并重新加载模板
func (mte *ModernTemplateEngine) checkAndReload(name string) error {
	mte.mu.RLock()
	tmpl, exists := mte.templates[name]
	mte.mu.RUnlock()
	
	if !exists || tmpl.Path == "" {
		return nil
	}
	
	stat, err := os.Stat(tmpl.Path)
	if err != nil {
		return nil // 文件可能被删除，忽略错误
	}
	
	modTime := stat.ModTime()
	lastModTime, exists := mte.watchers[tmpl.Path]
	
	if !exists || modTime.After(*lastModTime) {
		// 文件已修改，重新加载
		return mte.LoadTemplateFile(name, tmpl.Path)
	}
	
	return nil
}

// RegisterFunction 注册模板函数
func (mte *ModernTemplateEngine) RegisterFunction(name string, fn interface{}) error {
	mte.mu.Lock()
	defer mte.mu.Unlock()
	
	// 验证函数签名
	if err := mte.validateFunction(name, fn); err != nil {
		return err
	}
	
	mte.functions[name] = &TemplateFunction{
		Name:     name,
		Function: fn,
	}
	
	mte.funcMap[name] = fn
	
	// 重新编译所有模板以包含新函数
	return mte.recompileAllTemplates()
}

// validateFunction 验证函数签名
func (mte *ModernTemplateEngine) validateFunction(name string, fn interface{}) error {
	if fn == nil {
		return fmt.Errorf("function %s cannot be nil", name)
	}
	
	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func {
		return fmt.Errorf("function %s must be a function, got %s", name, fnType.Kind())
	}
	
	// 检查返回值数量（最多2个，第二个必须是error）
	numOut := fnType.NumOut()
	if numOut > 2 {
		return fmt.Errorf("function %s can have at most 2 return values", name)
	}
	
	if numOut == 2 {
		errorType := reflect.TypeOf((*error)(nil)).Elem()
		if !fnType.Out(1).Implements(errorType) {
			return fmt.Errorf("function %s second return value must be error", name)
		}
	}
	
	return nil
}

// recompileAllTemplates 重新编译所有模板
func (mte *ModernTemplateEngine) recompileAllTemplates() error {
	for _, tmpl := range mte.templates {
		if err := mte.compileTemplate(tmpl); err != nil {
			return fmt.Errorf("failed to recompile template %s: %w", tmpl.Name, err)
		}
	}
	return nil
}

// ValidateTemplate 验证模板
func (mte *ModernTemplateEngine) ValidateTemplate(name string) error {
	mte.mu.RLock()
	tmpl, exists := mte.templates[name]
	mte.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("template %s not found", name)
	}
	
	result := mte.validator.Validate(tmpl)
	if !result.IsValid {
		var errorMsgs []string
		for _, err := range result.Errors {
			errorMsgs = append(errorMsgs, fmt.Sprintf("Line %d: %s", err.Line, err.Message))
		}
		return fmt.Errorf("template validation failed:\n%s", strings.Join(errorMsgs, "\n"))
	}
	
	tmpl.Validated = true
	return nil
}

// GetTemplate 获取模板
func (mte *ModernTemplateEngine) GetTemplate(name string) (*Template, error) {
	mte.mu.RLock()
	defer mte.mu.RUnlock()
	
	tmpl, exists := mte.templates[name]
	if !exists {
		return nil, fmt.Errorf("template %s not found", name)
	}
	
	// 返回副本以避免并发修改
	copy := *tmpl
	return &copy, nil
}

// ListTemplates 列出所有模板
func (mte *ModernTemplateEngine) ListTemplates() []string {
	mte.mu.RLock()
	defer mte.mu.RUnlock()
	
	names := make([]string, 0, len(mte.templates))
	for name := range mte.templates {
		names = append(names, name)
	}
	return names
}

// ReloadTemplates 重新加载所有模板
func (mte *ModernTemplateEngine) ReloadTemplates() error {
	mte.mu.Lock()
	defer mte.mu.Unlock()
	
	for name, tmpl := range mte.templates {
		if tmpl.Path != "" {
			if err := mte.LoadTemplateFile(name, tmpl.Path); err != nil {
				return fmt.Errorf("failed to reload template %s: %w", name, err)
			}
		}
	}
	
	return nil
}

// NewTemplateValidator 创建模板验证器
func NewTemplateValidator() *TemplateValidator {
	validator := &TemplateValidator{
		rules: make([]ValidationRule, 0),
	}
	
	// 添加默认验证规则
	validator.addDefaultRules()
	
	return validator
}

// addDefaultRules 添加默认验证规则
func (tv *TemplateValidator) addDefaultRules() {
	// 检查未关闭的模板标签
	tv.rules = append(tv.rules, ValidationRule{
		Name:       "unclosed_tags",
		Pattern:    regexp.MustCompile(`{{\s*[^}]*$`),
		Severity:   SeverityError,
		Message:    "Unclosed template tag",
		Suggestion: "Ensure all template tags are properly closed with }}",
	})
	
	// 检查未定义的变量
	tv.rules = append(tv.rules, ValidationRule{
		Name:       "undefined_variables",
		Pattern:    regexp.MustCompile(`{{\s*\.(\w+)`),
		Severity:   SeverityWarning,
		Message:    "Potentially undefined variable",
		Suggestion: "Ensure the variable is defined in the template data",
	})
	
	// 检查模板函数调用
	tv.rules = append(tv.rules, ValidationRule{
		Name:       "function_calls",
		Pattern:    regexp.MustCompile(`{{\s*(\w+)\s+`),
		Severity:   SeverityInfo,
		Message:    "Template function call",
		Suggestion: "Verify that the function is registered",
	})
}

// Validate 验证模板
func (tv *TemplateValidator) Validate(tmpl *Template) ValidationResult {
	result := ValidationResult{
		IsValid:     true,
		Errors:      make([]TemplateError, 0),
		Warnings:    make([]TemplateError, 0),
		Suggestions: make([]string, 0),
		Metadata:    make(map[string]interface{}),
	}
	
	lines := strings.Split(tmpl.Content, "\n")
	
	for lineNum, line := range lines {
		for _, rule := range tv.rules {
			if rule.Pattern.MatchString(line) {
				templateErr := TemplateError{
					Line:       lineNum + 1,
					Column:     0, // 可以进一步优化以获取准确的列位置
					Code:       ErrTemplateCompilation,
					Severity:   rule.Severity,
					Message:    rule.Message,
					Suggestion: rule.Suggestion,
				}
				
				switch rule.Severity {
				case SeverityError, SeverityCritical:
					result.Errors = append(result.Errors, templateErr)
					result.IsValid = false
				case SeverityWarning:
					result.Warnings = append(result.Warnings, templateErr)
				}
			}
		}
	}
	
	// 添加通用建议
	if len(result.Errors) == 0 && len(result.Warnings) == 0 {
		result.Suggestions = append(result.Suggestions, "Template looks good!")
	}
	
	return result
}

// TemplateRegistry 模板注册中心
type TemplateRegistry struct {
	engines map[string]TemplateEngine
	mu      sync.RWMutex
}

// NewTemplateRegistry 创建模板注册中心
func NewTemplateRegistry() *TemplateRegistry {
	return &TemplateRegistry{
		engines: make(map[string]TemplateEngine),
	}
}

// RegisterEngine 注册模板引擎
func (tr *TemplateRegistry) RegisterEngine(name string, engine TemplateEngine) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.engines[name] = engine
}

// GetEngine 获取模板引擎
func (tr *TemplateRegistry) GetEngine(name string) (TemplateEngine, error) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	
	engine, exists := tr.engines[name]
	if !exists {
		return nil, fmt.Errorf("template engine %s not found", name)
	}
	
	return engine, nil
}

// ListEngines 列出所有引擎
func (tr *TemplateRegistry) ListEngines() []string {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	
	names := make([]string, 0, len(tr.engines))
	for name := range tr.engines {
		names = append(names, name)
	}
	return names 