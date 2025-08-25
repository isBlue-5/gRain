package processor

import (
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/isBlue-5/grain/pkg/annotation/registry"
	"github.com/isBlue-5/grain/pkg/annotation/types"
)

// RateLimitProcessor 限流注解处理器
type RateLimitProcessor struct {
	registry   registry.Registry
	parser     AnnotationParser
	outputPath string
}

// NewRateLimitProcessor 创建限流注解处理器
func NewRateLimitProcessor(registry registry.Registry, parser AnnotationParser, outputPath string) *RateLimitProcessor {
	return &RateLimitProcessor{
		registry:   registry,
		parser:     parser,
		outputPath: outputPath,
	}
}

// ProcessRateLimit 处理限流注解
func (p *RateLimitProcessor) ProcessRateLimit(paths []string) error {
	for _, path := range paths {
		if err := p.processPath(path); err != nil {
			return fmt.Errorf("处理路径 %s 失败: %w", path, err)
		}
	}
	return nil
}

// processPath 处理单个路径
func (p *RateLimitProcessor) processPath(path string) error {
	// 解析注解
	annotations, err := p.parser.ParseFile(path)
	if err != nil {
		return fmt.Errorf("解析注解失败: %w", err)
	}

	// 按包分组处理
	packageAnnotations := make(map[string][]types.CommentAnnotation)
	for _, ann := range annotations {
		if commentAnn, ok := ann.(*types.CommentAnnotation); ok &&
			(commentAnn.Type == types.RateLimitType || commentAnn.Type == "ratelimit") {
			pkgName := p.extractPackageName(path)
			packageAnnotations[pkgName] = append(packageAnnotations[pkgName], *commentAnn)
		}
	}

	// 处理每个包的注解
	for pkgName, anns := range packageAnnotations {
		if err := p.processPackage(pkgName, anns); err != nil {
			return fmt.Errorf("处理包 %s 失败: %w", pkgName, err)
		}
	}

	return nil
}

// processPackage 处理单个包的限流注解
func (p *RateLimitProcessor) processPackage(pkgName string, annotations []types.CommentAnnotation) error {
	for _, ann := range annotations {
		if ann.TargetType == types.MethodTarget {
			if err := p.processMethodAnnotation(pkgName, ann); err != nil {
				return err
			}
		} else if ann.TargetType == types.FunctionTarget {
			if err := p.processFunctionAnnotation(pkgName, ann); err != nil {
				return err
			}
		}
	}
	return nil
}

// processMethodAnnotation 处理方法上的限流注解
func (p *RateLimitProcessor) processMethodAnnotation(pkgName string, ann types.CommentAnnotation) error {
	// 提取注解参数
	limit := types.GetIntAttribute(ann.Attributes, "limit", 100)
	period := types.GetStringAttribute(ann.Attributes, "period", "1m")
	key := types.GetStringAttribute(ann.Attributes, "key", "ip")
	algorithm := types.GetStringAttribute(ann.Attributes, "algorithm", "sliding_window")
	errorCode := types.GetIntAttribute(ann.Attributes, "errorCode", 429)
	errorMessage := types.GetStringAttribute(ann.Attributes, "errorMessage", "Rate limit exceeded")

	// 分析方法
	methodDecl, ok := ann.Node.(*ast.FuncDecl)
	if !ok {
		return fmt.Errorf("限流注解只能应用于方法或函数: %s", ann.TargetName)
	}

	// 提取方法签名信息
	signatureInfo, err := p.extractMethodSignature(methodDecl)
	if err != nil {
		return fmt.Errorf("提取方法签名失败: %w", err)
	}

	// 获取原始方法名和包装方法名
	originalName := methodDecl.Name.Name
	wrappedName := originalName
	originalName = originalName + "Impl" // 原始方法将被重命名

	// 创建模板数据
	tmplData := map[string]interface{}{
		"Package":      pkgName,
		"WrappedName":  wrappedName,
		"OriginalName": originalName,
		"Limit":        limit,
		"Period":       period,
		"Key":          key,
		"Algorithm":    algorithm,
		"ErrorCode":    errorCode,
		"ErrorMessage": errorMessage,
		"LimiterField": "rateLimiter", // 限流器字段名
		"KeyGenerator": p.generateKeyGenerator(key),
	}

	// 合并方法签名信息
	for k, v := range signatureInfo {
		tmplData[k] = v
	}

	// 生成包装代码文件
	outputFile := filepath.Join(p.outputPath, "ratelimit", fmt.Sprintf("%s_ratelimit.go", strings.ToLower(ann.TargetName)))

	// 创建输出目录
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 创建输出文件
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer file.Close()

	// 生成代码
	if err := p.generateRateLimitCode(file, tmplData); err != nil {
		return fmt.Errorf("生成限流代码失败: %w", err)
	}

	return nil
}

// processFunctionAnnotation 处理函数上的限流注解
func (p *RateLimitProcessor) processFunctionAnnotation(pkgName string, ann types.CommentAnnotation) error {
	// 函数注解处理逻辑类似方法注解
	return p.processMethodAnnotation(pkgName, ann)
}

// extractMethodSignature 提取方法签名信息
func (p *RateLimitProcessor) extractMethodSignature(methodDecl *ast.FuncDecl) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// 提取接收者信息
	if methodDecl.Recv != nil && len(methodDecl.Recv.List) > 0 {
		recv := methodDecl.Recv.List[0]
		if starExpr, ok := recv.Type.(*ast.StarExpr); ok {
			if ident, ok := starExpr.X.(*ast.Ident); ok {
				result["ReceiverType"] = ident.Name
			}
		} else if ident, ok := recv.Type.(*ast.Ident); ok {
			result["ReceiverType"] = ident.Name
		}
	}

	// 提取参数信息
	if methodDecl.Type.Params != nil {
		params := make([]string, 0)
		paramNames := make([]string, 0)
		paramTypes := make([]string, 0)

		for _, param := range methodDecl.Type.Params.List {
			paramType := p.extractTypeString(param.Type)
			params = append(params, paramType)

			// 提取参数名
			for _, name := range param.Names {
				paramNames = append(paramNames, name.Name)
				paramTypes = append(paramTypes, paramType)
			}
		}

		result["Params"] = params
		result["ParamNames"] = paramNames
		result["ParamTypes"] = paramTypes
	}

	// 提取返回值信息
	if methodDecl.Type.Results != nil {
		returns := make([]string, 0)
		for _, result := range methodDecl.Type.Results.List {
			returnType := p.extractTypeString(result.Type)
			returns = append(returns, returnType)
		}
		result["Returns"] = returns
	}

	return result, nil
}

// extractTypeString 提取类型字符串
func (p *RateLimitProcessor) extractTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + p.extractTypeString(t.X)
	case *ast.SelectorExpr:
		return p.extractTypeString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + p.extractTypeString(t.Elt)
	case *ast.InterfaceType:
		return "interface{}"
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// generateKeyGenerator 生成键生成器代码
func (p *RateLimitProcessor) generateKeyGenerator(key string) string {
	switch key {
	case "ip":
		return "c.ClientIP()"
	case "path":
		return "c.FullPath()"
	case "user":
		return "getUserID(c)"
	case "header":
		return "c.GetHeader(\"X-User-ID\")"
	default:
		if strings.HasPrefix(key, "header:") {
			headerName := strings.TrimPrefix(key, "header:")
			return fmt.Sprintf("c.GetHeader(\"%s\")", headerName)
		}
		return fmt.Sprintf("\"%s\"", key)
	}
}

// generateRateLimitCode 生成限流代码
func (p *RateLimitProcessor) generateRateLimitCode(file *os.File, data map[string]interface{}) error {
	const rateLimitTemplate = `// Code generated by gRain framework. DO NOT EDIT.

package {{.Package}}

import (
	"net/http"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/isBlue-5/grain/pkg/annotation/processor"
)

// {{.WrappedName}} 限流包装方法
func ({{if .ReceiverType}}r *{{.ReceiverType}}{{end}}) {{.WrappedName}}(c *gin.Context{{if .ParamNames}}, {{range $i, $name := .ParamNames}}{{$name}} {{index $.ParamTypes $i}}{{if ne $i (sub (len $.ParamNames) 1)}}, {{end}}{{end}}{{end}}) {{if .Returns}}({{range $i, $ret := .Returns}}{{$ret}}{{if ne $i (sub (len $.Returns) 1)}}, {{end}}{{end}}){{end}} {
	// 获取限流键
	limiterKey := {{.KeyGenerator}}
	
	// 检查限流
	if !checkRateLimit("{{.WrappedName}}", {{.Limit}}, "{{.Period}}", "{{.Algorithm}}", limiterKey) {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"error": "{{.ErrorMessage}}",
			"limit": {{.Limit}},
			"period": "{{.Period}}",
		})
		return{{if .Returns}} {{range $i, $ret := .Returns}}{{if eq $ret "error"}}fmt.Errorf("rate limit exceeded"){{else}}nil{{end}}{{if ne $i (sub (len $.Returns) 1)}}, {{end}}{{end}}{{end}}
	}
	
	// 调用原始方法
	return {{if .ReceiverType}}r.{{end}}{{.OriginalName}}(c{{if .ParamNames}}, {{range $i, $name := .ParamNames}}{{$name}}{{if ne $i (sub (len $.ParamNames) 1)}}, {{end}}{{end}}{{end}})
}

// getUserID 获取用户ID的辅助函数
func getUserID(c *gin.Context) string {
	if user, exists := c.Get("user"); exists {
		if userID, ok := user.(string); ok {
			return userID
		}
	}
	return "anonymous"
}

// checkRateLimit 检查限流的辅助函数
func checkRateLimit(methodName string, limit int, period string, algorithm string, key string) bool {
	// 获取全局限流管理器
	manager := GetGlobalRateLimitManager()
	if manager == nil {
		// 如果限流管理器未初始化，记录警告并允许访问
		log.Printf("Warning: RateLimitManager not initialized, allowing access for method %s", methodName)
		return true
	}
	
	// 调用真实的限流检查逻辑
	return manager.Allow(methodName, key, limit, period, algorithm)
}
`

	tmpl, err := template.New("rateLimit").Funcs(template.FuncMap{
		"sub": func(a, b int) int {
			return a - b
		},
	}).Parse(rateLimitTemplate)
	if err != nil {
		return err
	}

	return tmpl.Execute(file, data)
}

// extractPackageName 提取包名
func (p *RateLimitProcessor) extractPackageName(path string) string {
	// 使用共享的包名生成器
	return RateLimitPackageGenerator.GetPackageName(path)
}
