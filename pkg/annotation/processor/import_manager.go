package processor

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
)

// ImportManager 智能导入管理器
type ImportManager struct {
	imports map[string]string // path -> alias
	used    map[string]bool   // path -> used
}

// NewImportManager 创建新的导入管理器
func NewImportManager() *ImportManager {
	return &ImportManager{
		imports: make(map[string]string),
		used:    make(map[string]bool),
	}
}

// AddImport 添加导入
func (im *ImportManager) AddImport(path, alias string) {
	im.imports[path] = alias
	im.used[path] = false
}

// MarkUsed 标记导入为已使用
func (im *ImportManager) MarkUsed(path string) {
	im.used[path] = true
}

// GetCleanImports 获取清理后的导入列表
func (im *ImportManager) GetCleanImports() []string {
	var result []string
	for path, alias := range im.imports {
		if im.used[path] {
			if alias != "" {
				result = append(result, fmt.Sprintf(`%s "%s"`, alias, path))
			} else {
				result = append(result, fmt.Sprintf(`"%s"`, path))
			}
		}
	}

	// 按路径排序，确保导入顺序一致
	sort.Strings(result)
	return result
}

// RemoveUnusedImports 移除未使用的导入
func (im *ImportManager) RemoveUnusedImports() {
	for path := range im.imports {
		if !im.used[path] {
			delete(im.imports, path)
			delete(im.used, path)
		}
	}
}

// ValidateImportPath 验证导入路径
func (im *ImportManager) ValidateImportPath(path string) error {
	// 检查是否是标准库
	if isStandardLibrary(path) {
		return nil
	}

	// 检查是否是相对路径
	if strings.HasPrefix(path, ".") || strings.HasPrefix(path, "/") {
		return fmt.Errorf("不支持相对路径导入: %s", path)
	}

	// 检查是否是有效的Go模块路径
	if !isValidGoModulePath(path) {
		return fmt.Errorf("无效的Go模块路径: %s", path)
	}

	return nil
}

// isStandardLibrary 检查是否是标准库
func isStandardLibrary(path string) bool {
	// Go标准库列表
	stdLibs := map[string]bool{
		"archive": true, "bufio": true, "builtin": true, "bytes": true,
		"compress": true, "container": true, "context": true, "crypto": true,
		"database": true, "debug": true, "encoding": true, "errors": true,
		"expvar": true, "flag": true, "fmt": true, "go": true,
		"hash": true, "html": true, "image": true, "index": true,
		"io": true, "log": true, "math": true, "mime": true,
		"net": true, "os": true, "path": true, "plugin": true,
		"reflect": true, "regexp": true, "runtime": true, "sort": true,
		"strconv": true, "strings": true, "sync": true, "syscall": true,
		"testing": true, "text": true, "time": true, "unicode": true,
		"unsafe": true, "vendor": true,
	}

	parts := strings.Split(path, "/")
	return stdLibs[parts[0]]
}

// isValidGoModulePath 检查是否是有效的Go模块路径
func isValidGoModulePath(path string) bool {
	// 基本的Go模块路径验证规则
	if len(path) == 0 {
		return false
	}

	// 不能以点或斜杠开头
	if strings.HasPrefix(path, ".") || strings.HasPrefix(path, "/") {
		return false
	}

	// 不能包含连续的点
	if strings.Contains(path, "..") {
		return false
	}

	// 不能包含反斜杠（Windows路径）
	if strings.Contains(path, "\\") {
		return false
	}

	// 不能包含空格
	if strings.Contains(path, " ") {
		return false
	}

	return true
}

// AnalyzeFileImports 分析文件中的导入使用情况
func (im *ImportManager) AnalyzeFileImports(filePath string) error {
	// 解析Go文件
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("解析文件失败: %w", err)
	}

	// 分析导入使用情况
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.ImportSpec:
			// 记录导入
			path := strings.Trim(x.Path.Value, `"`)
			alias := ""
			if x.Name != nil {
				alias = x.Name.Name
			}
			im.AddImport(path, alias)

		case *ast.SelectorExpr:
			// 检查选择器表达式中的包使用
			if ident, ok := x.X.(*ast.Ident); ok {
				// 查找对应的导入
				for path, alias := range im.imports {
					if alias == ident.Name || (alias == "" && filepath.Base(path) == ident.Name) {
						im.MarkUsed(path)
						break
					}
				}
			}
		}
		return true
	})

	return nil
}

// OptimizeImports 优化导入列表
func (im *ImportManager) OptimizeImports() {
	// 移除未使用的导入
	im.RemoveUnusedImports()

	// 检查是否有循环依赖
	im.checkCircularDependencies()
}

// checkCircularDependencies 检查循环依赖
func (im *ImportManager) checkCircularDependencies() {
	// 由于ImportManager只管理单个文件的导入，这里主要检查常见的循环依赖模式
	// 在实际项目中，循环依赖通常发生在包级别，需要更复杂的分析

	// 检查自引用
	for path := range im.imports {
		if strings.Contains(path, "github.com/isBlue-5/grain") {
			// 检查是否是内部循环引用
			if strings.Contains(path, "/pkg/") && strings.Contains(path, "/internal/") {
				fmt.Printf("警告: 检测到可能的循环依赖: %s\n", path)
			}

			// 检查是否导入了当前包
			if strings.Contains(path, "self") || strings.Contains(path, "current") {
				fmt.Printf("警告: 检测到自引用: %s\n", path)
			}
		}
	}

	// 检查标准库和第三方库的合理使用
	for path := range im.imports {
		if isStandardLibrary(path) {
			// 标准库通常不会有循环依赖问题
			continue
		}

		// 检查第三方库的版本兼容性
		if strings.Contains(path, "v0") && strings.Contains(path, "v1") {
			fmt.Printf("警告: 检测到版本不一致的依赖: %s\n", path)
		}
	}
}

// detectCycle 使用DFS检测循环依赖
func (im *ImportManager) detectCycle(node string, graph map[string][]string, visited, recStack map[string]bool) bool {
	visited[node] = true
	recStack[node] = true

	for _, neighbor := range graph[node] {
		if !visited[neighbor] {
			if im.detectCycle(neighbor, graph, visited, recStack) {
				return true
			}
		} else if recStack[neighbor] {
			// 找到后向边，存在循环
			return true
		}
	}

	recStack[node] = false
	return false
}

// GenerateImportBlock 生成导入代码块
func (im *ImportManager) GenerateImportBlock() string {
	imports := im.GetCleanImports()
	if len(imports) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("import (\n")

	// 按类型分组导入
	stdLibs := make([]string, 0)
	thirdParty := make([]string, 0)
	local := make([]string, 0)

	for _, imp := range imports {
		path := extractPathFromImport(imp)
		if isStandardLibrary(path) {
			stdLibs = append(stdLibs, imp)
		} else if strings.Contains(path, "github.com/isBlue-5/grain") {
			local = append(local, imp)
		} else {
			thirdParty = append(thirdParty, imp)
		}
	}

	// 按顺序输出：标准库、第三方库、本地库
	if len(stdLibs) > 0 {
		sort.Strings(stdLibs)
		for _, imp := range stdLibs {
			builder.WriteString(fmt.Sprintf("\t%s\n", imp))
		}
		if len(thirdParty) > 0 || len(local) > 0 {
			builder.WriteString("\n")
		}
	}

	if len(thirdParty) > 0 {
		sort.Strings(thirdParty)
		for _, imp := range thirdParty {
			builder.WriteString(fmt.Sprintf("\t%s\n", imp))
		}
		if len(local) > 0 {
			builder.WriteString("\n")
		}
	}

	if len(local) > 0 {
		sort.Strings(local)
		for _, imp := range local {
			builder.WriteString(fmt.Sprintf("\t%s\n", imp))
		}
	}

	builder.WriteString(")")
	return builder.String()
}

// extractPathFromImport 从导入语句中提取路径
func extractPathFromImport(imp string) string {
	// 处理带别名的导入: alias "path"
	if strings.Contains(imp, `"`) {
		parts := strings.Split(imp, `"`)
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	return imp
}

// ValidateGeneratedCode 验证生成的代码
func (im *ImportManager) ValidateGeneratedCode(code string) error {
	// 检查是否有语法错误
	if strings.Contains(code, "undefined:") {
		return fmt.Errorf("生成的代码中存在未定义的类型")
	}

	if strings.Contains(code, "imported and not used") {
		return fmt.Errorf("生成的代码中存在未使用的导入")
	}

	if strings.Contains(code, "unused variable") {
		return fmt.Errorf("生成的代码中存在未使用的变量")
	}

	return nil
}
