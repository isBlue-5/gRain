package interfaces

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// InterfaceAnalysis 接口分析结果
type InterfaceAnalysis struct {
	Name            string
	Methods         []MethodInfo
	Complexity      int
	Cohesion        float64
	Coupling        float64
	Maintainability float64
}

// MethodInfo 方法信息
type MethodInfo struct {
	Name       string
	Params     []ParamInfo
	Returns    []ParamInfo
	Complexity int
}

// ParamInfo 参数信息
type ParamInfo struct {
	Name string
	Type string
}

// Optimization 优化建议
type Optimization struct {
	Type        string
	Description string
	Priority    int
	Impact      string
	Params      map[string]interface{} // 优化参数
}

// OptimizationResult 优化结果
type OptimizationResult struct {
	OriginalCode   string
	RefactoredCode string
	Documentation  string
	Optimizations  []Optimization
	Metrics        *InterfaceAnalysis
}

// InterfaceAnalyzer 接口分析器
type InterfaceAnalyzer struct{}

// InterfaceRefactorer 接口重构器
type InterfaceRefactorer struct{}

// InterfaceValidator 接口验证器
type InterfaceValidator struct{}

// InterfaceOptimizer 接口设计优化器
type InterfaceOptimizer struct {
	// 接口分析器
	analyzer *InterfaceAnalyzer

	// 接口重构器
	refactorer *InterfaceRefactorer

	// 接口验证器
	validator *InterfaceValidator

	// 优化选项
	options *OptimizerOptions
}

// OptimizerOptions 优化器选项
type OptimizerOptions struct {
	// 是否启用接口合并
	EnableInterfaceMerging bool

	// 是否启用接口分离
	EnableInterfaceSegregation bool

	// 是否启用接口抽象
	EnableInterfaceAbstraction bool

	// 是否启用接口文档生成
	EnableDocumentation bool

	// 最大接口方法数
	MaxInterfaceMethods int

	// 是否启用接口测试生成
	EnableTestGeneration bool
}

// DefaultOptimizerOptions 默认优化器选项
func DefaultOptimizerOptions() *OptimizerOptions {
	return &OptimizerOptions{
		EnableInterfaceMerging:     true,
		EnableInterfaceSegregation: true,
		EnableInterfaceAbstraction: true,
		EnableDocumentation:        true,
		MaxInterfaceMethods:        10,
		EnableTestGeneration:       false,
	}
}

// NewInterfaceAnalyzer 创建新的接口分析器
func NewInterfaceAnalyzer() *InterfaceAnalyzer {
	return &InterfaceAnalyzer{}
}

// NewInterfaceRefactorer 创建新的接口重构器
func NewInterfaceRefactorer() *InterfaceRefactorer {
	return &InterfaceRefactorer{}
}

// NewInterfaceValidator 创建新的接口验证器
func NewInterfaceValidator() *InterfaceValidator {
	return &InterfaceValidator{}
}

// NewInterfaceOptimizer 创建新的接口设计优化器
func NewInterfaceOptimizer(options *OptimizerOptions) *InterfaceOptimizer {
	if options == nil {
		options = DefaultOptimizerOptions()
	}

	return &InterfaceOptimizer{
		analyzer:   NewInterfaceAnalyzer(),
		refactorer: NewInterfaceRefactorer(),
		validator:  NewInterfaceValidator(),
		options:    options,
	}
}

// OptimizeInterface 优化接口设计
func (io *InterfaceOptimizer) OptimizeInterface(interfaceName string, sourceCode string) (*OptimizationResult, error) {
	// 1. 分析接口
	analysis, err := io.analyzer.AnalyzeInterface(sourceCode, interfaceName)
	if err != nil {
		return nil, fmt.Errorf("interface analysis failed: %w", err)
	}

	// 2. 验证接口
	if err := io.validator.ValidateInterface(analysis); err != nil {
		return nil, fmt.Errorf("interface validation failed: %w", err)
	}

	// 3. 应用优化策略
	optimizations := io.applyOptimizationStrategies(analysis)

	// 4. 重构接口
	refactoredCode, err := io.refactorer.RefactorInterface(sourceCode, optimizations)
	if err != nil {
		return nil, fmt.Errorf("interface refactoring failed: %w", err)
	}

	// 5. 生成文档
	var documentation string
	if io.options.EnableDocumentation {
		documentation = io.generateDocumentation(analysis, optimizations)
	}

	return &OptimizationResult{
		OriginalCode:   sourceCode,
		RefactoredCode: refactoredCode,
		Documentation:  documentation,
		Optimizations:  optimizations,
		Metrics:        analysis,
	}, nil
}

// AnalyzeInterface 分析接口
func (ia *InterfaceAnalyzer) AnalyzeInterface(sourceCode, interfaceName string) (*InterfaceAnalysis, error) {
	// 解析Go代码
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", sourceCode, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source code: %w", err)
	}

	// 查找接口定义
	var interfaceDecl *ast.InterfaceType
	ast.Inspect(node, func(n ast.Node) bool {
		if typeDecl, ok := n.(*ast.TypeSpec); ok {
			if typeDecl.Name.Name == interfaceName {
				if interfaceType, ok := typeDecl.Type.(*ast.InterfaceType); ok {
					interfaceDecl = interfaceType
					return false
				}
			}
		}
		return true
	})

	if interfaceDecl == nil {
		return nil, fmt.Errorf("interface %s not found", interfaceName)
	}

	// 分析接口方法
	methods := make([]MethodInfo, 0)
	for _, method := range interfaceDecl.Methods.List {
		if funcType, ok := method.Type.(*ast.FuncType); ok {
			methodInfo := MethodInfo{
				Name:       method.Names[0].Name,
				Params:     ia.analyzeParams(funcType.Params),
				Returns:    ia.analyzeParams(funcType.Results),
				Complexity: ia.calculateMethodComplexity(funcType),
			}
			methods = append(methods, methodInfo)
		}
	}

	// 计算接口指标
	complexity := ia.calculateInterfaceComplexity(methods)
	cohesion := ia.calculateCohesion(methods)
	coupling := ia.calculateCoupling(methods)
	maintainability := ia.calculateMaintainability(complexity, cohesion, coupling)

	return &InterfaceAnalysis{
		Name:            interfaceName,
		Methods:         methods,
		Complexity:      complexity,
		Cohesion:        cohesion,
		Coupling:        coupling,
		Maintainability: maintainability,
	}, nil
}

// analyzeParams 分析参数
func (ia *InterfaceAnalyzer) analyzeParams(fieldList *ast.FieldList) []ParamInfo {
	if fieldList == nil {
		return nil
	}

	params := make([]ParamInfo, 0)
	for _, field := range fieldList.List {
		paramType := ia.getTypeString(field.Type)
		if len(field.Names) > 0 {
			for _, name := range field.Names {
				params = append(params, ParamInfo{
					Name: name.Name,
					Type: paramType,
				})
			}
		} else {
			params = append(params, ParamInfo{
				Name: "",
				Type: paramType,
			})
		}
	}
	return params
}

// getTypeString 获取类型字符串
func (ia *InterfaceAnalyzer) getTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + ia.getTypeString(t.X)
	case *ast.ArrayType:
		return "[]" + ia.getTypeString(t.Elt)
	case *ast.SelectorExpr:
		return ia.getTypeString(t.X) + "." + t.Sel.Name
	default:
		return "unknown"
	}
}

// calculateMethodComplexity 计算方法复杂度
func (ia *InterfaceAnalyzer) calculateMethodComplexity(funcType *ast.FuncType) int {
	complexity := 1 // 基础复杂度

	// 参数复杂度
	if funcType.Params != nil {
		complexity += len(funcType.Params.List)
	}

	// 返回值复杂度
	if funcType.Results != nil {
		complexity += len(funcType.Results.List)
	}

	return complexity
}

// calculateInterfaceComplexity 计算接口复杂度
func (ia *InterfaceAnalyzer) calculateInterfaceComplexity(methods []MethodInfo) int {
	totalComplexity := 0
	for _, method := range methods {
		totalComplexity += method.Complexity
	}
	return totalComplexity
}

// calculateCohesion 计算内聚性
func (ia *InterfaceAnalyzer) calculateCohesion(methods []MethodInfo) float64 {
	if len(methods) <= 1 {
		return 1.0
	}

	// 简单的内聚性计算：基于方法名称的相似性
	similarityCount := 0
	totalComparisons := 0

	for i := 0; i < len(methods); i++ {
		for j := i + 1; j < len(methods); j++ {
			totalComparisons++
			if ia.areMethodsSimilar(methods[i], methods[j]) {
				similarityCount++
			}
		}
	}

	if totalComparisons == 0 {
		return 1.0
	}

	return float64(similarityCount) / float64(totalComparisons)
}

// areMethodsSimilar 判断方法是否相似
func (ia *InterfaceAnalyzer) areMethodsSimilar(m1, m2 MethodInfo) bool {
	// 简单的相似性判断：基于方法名称前缀
	prefix1 := ia.getMethodPrefix(m1.Name)
	prefix2 := ia.getMethodPrefix(m2.Name)
	return prefix1 == prefix2
}

// getMethodPrefix 获取方法前缀
func (ia *InterfaceAnalyzer) getMethodPrefix(methodName string) string {
	// 提取方法名称的前缀（如Get、Set、Is等）
	words := strings.FieldsFunc(methodName, func(r rune) bool {
		return r >= 'A' && r <= 'Z'
	})
	if len(words) > 0 {
		return words[0]
	}
	return methodName
}

// calculateCoupling 计算耦合性
func (ia *InterfaceAnalyzer) calculateCoupling(methods []MethodInfo) float64 {
	if len(methods) == 0 {
		return 0.0
	}

	// 简单的耦合性计算：基于参数类型的多样性
	typeCount := make(map[string]int)
	totalParams := 0

	for _, method := range methods {
		for _, param := range method.Params {
			typeCount[param.Type]++
			totalParams++
		}
	}

	if totalParams == 0 {
		return 0.0
	}

	// 耦合性 = 1 - (唯一类型数 / 总参数数)
	uniqueTypes := len(typeCount)
	return 1.0 - float64(uniqueTypes)/float64(totalParams)
}

// calculateMaintainability 计算可维护性
func (ia *InterfaceAnalyzer) calculateMaintainability(complexity int, cohesion, coupling float64) float64 {
	// 可维护性 = 内聚性 * (1 - 耦合性) / 复杂度
	if complexity == 0 {
		complexity = 1
	}

	maintainability := cohesion * (1 - coupling) / float64(complexity)

	// 确保值在0-1范围内
	if maintainability < 0 {
		maintainability = 0
	}
	if maintainability > 1 {
		maintainability = 1
	}

	return maintainability
}

// ValidateInterface 验证接口
func (iv *InterfaceValidator) ValidateInterface(analysis *InterfaceAnalysis) error {
	// 检查接口方法数量
	if len(analysis.Methods) > 20 {
		return fmt.Errorf("interface has too many methods: %d (max: 20)", len(analysis.Methods))
	}

	// 检查复杂度
	if analysis.Complexity > 100 {
		return fmt.Errorf("interface is too complex: %d (max: 100)", analysis.Complexity)
	}

	// 检查可维护性
	if analysis.Maintainability < 0.3 {
		return fmt.Errorf("interface has low maintainability: %.2f (min: 0.3)", analysis.Maintainability)
	}

	return nil
}

// RefactorInterface 重构接口
func (ir *InterfaceRefactorer) RefactorInterface(sourceCode string, optimizations []Optimization) (string, error) {
	if sourceCode == "" {
		return "", errors.New("source code cannot be empty")
	}

	if len(optimizations) == 0 {
		return sourceCode, nil // 没有优化策略，返回原始代码
	}

	// 应用每个优化策略
	refactoredCode := sourceCode
	for _, opt := range optimizations {
		var err error
		refactoredCode, err = ir.applyOptimization(refactoredCode, opt)
		if err != nil {
			return "", fmt.Errorf("failed to apply optimization %s: %w", opt.Type, err)
		}
	}

	return refactoredCode, nil
}

// applyOptimization 应用单个优化策略
func (ir *InterfaceRefactorer) applyOptimization(sourceCode string, opt Optimization) (string, error) {
	switch opt.Type {
	case "extract_interface":
		return ir.extractInterface(sourceCode, opt.Params)
	case "merge_interfaces":
		return ir.mergeInterfaces(sourceCode, opt.Params)
	case "split_interface":
		return ir.splitInterface(sourceCode, opt.Params)
	case "add_methods":
		return ir.addMethods(sourceCode, opt.Params)
	default:
		return sourceCode, fmt.Errorf("unknown optimization type: %s", opt.Type)
	}
}

// extractInterface 提取接口
func (ir *InterfaceRefactorer) extractInterface(sourceCode string, params map[string]interface{}) (string, error) {
	// 实现接口提取逻辑
	// 这里可以根据参数提取特定的方法组合成新接口
	return sourceCode, nil
}

// mergeInterfaces 合并接口
func (ir *InterfaceRefactorer) mergeInterfaces(sourceCode string, params map[string]interface{}) (string, error) {
	// 实现接口合并逻辑
	return sourceCode, nil
}

// splitInterface 拆分接口
func (ir *InterfaceRefactorer) splitInterface(sourceCode string, params map[string]interface{}) (string, error) {
	// 实现接口拆分逻辑
	return sourceCode, nil
}

// addMethods 添加方法
func (ir *InterfaceRefactorer) addMethods(sourceCode string, params map[string]interface{}) (string, error) {
	// 实现方法添加逻辑
	return sourceCode, nil
}

// applyOptimizationStrategies 应用优化策略
func (io *InterfaceOptimizer) applyOptimizationStrategies(analysis *InterfaceAnalysis) []Optimization {
	var optimizations []Optimization

	// 接口分离策略
	if io.options.EnableInterfaceSegregation && len(analysis.Methods) > io.options.MaxInterfaceMethods {
		optimizations = append(optimizations, Optimization{
			Type:        "Interface Segregation",
			Description: fmt.Sprintf("Interface has %d methods, consider splitting into smaller interfaces", len(analysis.Methods)),
			Priority:    1,
			Impact:      "High",
		})
	}

	// 复杂度优化策略
	if analysis.Complexity > 50 {
		optimizations = append(optimizations, Optimization{
			Type:        "Complexity Reduction",
			Description: fmt.Sprintf("Interface complexity is %d, consider simplifying", analysis.Complexity),
			Priority:    2,
			Impact:      "Medium",
		})
	}

	// 内聚性优化策略
	if analysis.Cohesion < 0.5 {
		optimizations = append(optimizations, Optimization{
			Type:        "Cohesion Improvement",
			Description: fmt.Sprintf("Interface cohesion is %.2f, consider grouping related methods", analysis.Cohesion),
			Priority:    3,
			Impact:      "Medium",
		})
	}

	// 耦合性优化策略
	if analysis.Coupling > 0.7 {
		optimizations = append(optimizations, Optimization{
			Type:        "Coupling Reduction",
			Description: fmt.Sprintf("Interface coupling is %.2f, consider reducing dependencies", analysis.Coupling),
			Priority:    2,
			Impact:      "High",
		})
	}

	return optimizations
}

// generateDocumentation 生成文档
func (io *InterfaceOptimizer) generateDocumentation(analysis *InterfaceAnalysis, optimizations []Optimization) string {
	var doc strings.Builder

	doc.WriteString(fmt.Sprintf("# Interface: %s\n\n", analysis.Name))
	doc.WriteString("## Metrics\n\n")
	doc.WriteString(fmt.Sprintf("- **Complexity**: %d\n", analysis.Complexity))
	doc.WriteString(fmt.Sprintf("- **Cohesion**: %.2f\n", analysis.Cohesion))
	doc.WriteString(fmt.Sprintf("- **Coupling**: %.2f\n", analysis.Coupling))
	doc.WriteString(fmt.Sprintf("- **Maintainability**: %.2f\n\n", analysis.Maintainability))

	doc.WriteString("## Methods\n\n")
	for _, method := range analysis.Methods {
		doc.WriteString(fmt.Sprintf("- **%s** (Complexity: %d)\n", method.Name, method.Complexity))
	}

	if len(optimizations) > 0 {
		doc.WriteString("\n## Optimization Suggestions\n\n")
		for _, opt := range optimizations {
			doc.WriteString(fmt.Sprintf("- **%s** (%s Priority, %s Impact): %s\n",
				opt.Type, getPriorityString(opt.Priority), opt.Impact, opt.Description))
		}
	}

	return doc.String()
}

// getPriorityString 获取优先级字符串
func getPriorityString(priority int) string {
	switch priority {
	case 1:
		return "High"
	case 2:
		return "Medium"
	case 3:
		return "Low"
	default:
		return "Unknown"
	}
}
