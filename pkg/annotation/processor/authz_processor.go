package processor

import (
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/grain-framework/grain/pkg/annotation/registry"
	"github.com/grain-framework/grain/pkg/annotation/types"
)

// AuthzProcessor 权限控制注解处理器
type AuthzProcessor struct {
	registry   registry.Registry
	parser     AnnotationParser
	outputPath string
}

// NewAuthzProcessor 创建权限控制注解处理器
func NewAuthzProcessor(registry registry.Registry, parser AnnotationParser, outputPath string) *AuthzProcessor {
	return &AuthzProcessor{
		registry:   registry,
		parser:     parser,
		outputPath: outputPath,
	}
}

// ProcessAuthz 处理权限控制注解
func (p *AuthzProcessor) ProcessAuthz(paths []string) error {
	for _, path := range paths {
		if err := p.processPath(path); err != nil {
			return fmt.Errorf("处理路径 %s 失败: %w", path, err)
		}
	}
	return nil
}

// processPath 处理单个路径
func (p *AuthzProcessor) processPath(path string) error {
	// 解析注解
	annotations, err := p.parser.ParseFile(path)
	if err != nil {
		return fmt.Errorf("解析注解失败: %w", err)
	}

	// 按包分组处理
	packageAnnotations := make(map[string][]types.CommentAnnotation)
	for _, ann := range annotations {
		if commentAnn, ok := ann.(*types.CommentAnnotation); ok && commentAnn.Type == types.AuthType {
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

// processPackage 处理单个包的权限控制注解
func (p *AuthzProcessor) processPackage(pkgName string, annotations []types.CommentAnnotation) error {
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

// processMethodAnnotation 处理方法上的权限控制注解
func (p *AuthzProcessor) processMethodAnnotation(pkgName string, ann types.CommentAnnotation) error {
	// 提取注解参数
	roles, _ := types.GetStringSliceAttribute(ann.Attributes, "roles")
	permissions, _ := types.GetStringSliceAttribute(ann.Attributes, "permissions")
	expression := types.GetStringAttribute(ann.Attributes, "expression", "")
	errorCode := types.GetIntAttribute(ann.Attributes, "errorCode", 403)
	errorMessage := types.GetStringAttribute(ann.Attributes, "errorMessage", "Access denied")

	// 分析方法
	methodDecl, ok := ann.Node.(*ast.FuncDecl)
	if !ok {
		return fmt.Errorf("权限控制注解只能应用于方法或函数: %s", ann.TargetName)
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
		"Package":         pkgName,
		"WrappedName":     wrappedName,
		"OriginalName":    originalName,
		"Roles":           roles,
		"Permissions":     permissions,
		"Expression":      expression,
		"ErrorCode":       errorCode,
		"ErrorMessage":    errorMessage,
		"AuthorizerField": "authorizer", // 授权器字段名
		"HasRoles":        len(roles) > 0,
		"HasPermissions":  len(permissions) > 0,
		"HasExpression":   expression != "",
	}

	// 合并方法签名信息
	for k, v := range signatureInfo {
		tmplData[k] = v
	}

	// 生成包装代码文件
	outputFile := filepath.Join(p.outputPath, fmt.Sprintf("%s_authz.go", strings.ToLower(ann.TargetName)))

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
	if err := p.generateAuthzCode(file, tmplData); err != nil {
		return fmt.Errorf("生成权限控制代码失败: %w", err)
	}

	return nil
}

// processFunctionAnnotation 处理函数上的权限控制注解
func (p *AuthzProcessor) processFunctionAnnotation(pkgName string, ann types.CommentAnnotation) error {
	// 函数注解处理逻辑类似方法注解
	return p.processMethodAnnotation(pkgName, ann)
}

// extractMethodSignature 提取方法签名信息
func (p *AuthzProcessor) extractMethodSignature(methodDecl *ast.FuncDecl) (map[string]interface{}, error) {
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
func (p *AuthzProcessor) extractTypeString(expr ast.Expr) string {
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

// generateAuthzCode 生成权限控制代码
func (p *AuthzProcessor) generateAuthzCode(file *os.File, data map[string]interface{}) error {
	const authzTemplate = `// Code generated by gRain framework. DO NOT EDIT.

package {{.Package}}

import (
	"net/http"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/grain-framework/grain/pkg/annotation/processor"
)

// {{.WrappedName}} 权限控制包装方法
func ({{if .ReceiverType}}r *{{.ReceiverType}}{{end}}) {{.WrappedName}}(c *gin.Context{{if .ParamNames}}, {{range $i, $name := .ParamNames}}{{$name}} {{index $.ParamTypes $i}}{{if ne $i (sub (len $.ParamNames) 1)}}, {{end}}{{end}}{{end}}) {{if .Returns}}({{range $i, $ret := .Returns}}{{$ret}}{{if ne $i (sub (len $.Returns) 1)}}, {{end}}{{end}}){{end}} {
	// 获取用户身份
	userID := getCurrentUserID(c)
	if userID == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
		})
		return{{if .Returns}} {{range $i, $ret := .Returns}}{{if eq $ret "error"}}fmt.Errorf("authentication required"){{else}}nil{{end}}{{if ne $i (sub (len $.Returns) 1)}}, {{end}}{{end}}{{end}}
	}

	// 权限检查
	{{if .HasRoles}}
	// 角色检查
	if !hasRequiredRoles(userID, []string{ {{range $i, $role := .Roles}}"{{$role}}"{{if ne $i (sub (len $.Roles) 1)}}, {{end}}{{end}} }) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "{{.ErrorMessage}}",
			"required_roles": []string{ {{range $i, $role := .Roles}}"{{$role}}"{{if ne $i (sub (len $.Roles) 1)}}, {{end}}{{end}} },
		})
		return{{if .Returns}} {{range $i, $ret := .Returns}}{{if eq $ret "error"}}fmt.Errorf("insufficient roles"){{else}}nil{{end}}{{if ne $i (sub (len $.Returns) 1)}}, {{end}}{{end}}{{end}}
	}
	{{end}}

	{{if .HasPermissions}}
	// 权限检查
	if !hasRequiredPermissions(userID, []string{ {{range $i, $perm := .Permissions}}"{{$perm}}"{{if ne $i (sub (len $.Permissions) 1)}}, {{end}}{{end}} }) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "{{.ErrorMessage}}",
			"required_permissions": []string{ {{range $i, $perm := .Permissions}}"{{$perm}}"{{if ne $i (sub (len $.Permissions) 1)}}, {{end}}{{end}} },
		})
		return{{if .Returns}} {{range $i, $ret := .Returns}}{{if eq $ret "error"}}fmt.Errorf("insufficient permissions"){{else}}nil{{end}}{{if ne $i (sub (len $.Returns) 1)}}, {{end}}{{end}}{{end}}
	}
	{{end}}

	{{if .HasExpression}}
	// 表达式权限检查
	if !evaluateExpression(userID, "{{.Expression}}") {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "{{.ErrorMessage}}",
			"expression": "{{.Expression}}",
		})
		return{{if .Returns}} {{range $i, $ret := .Returns}}{{if eq $ret "error"}}fmt.Errorf("expression evaluation failed"){{else}}nil{{end}}{{if ne $i (sub (len $.Returns) 1)}}, {{end}}{{end}}{{end}}
	}
	{{end}}

	// 调用原始方法
	return {{if .ReceiverType}}r.{{end}}{{.OriginalName}}(c{{if .ParamNames}}, {{range $i, $name := .ParamNames}}{{$name}}{{if ne $i (sub (len $.ParamNames) 1)}}, {{end}}{{end}}{{end}})
}

// 辅助函数实现
func getCurrentUserID(c *gin.Context) string {
	// 从上下文中获取用户ID
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(string); ok {
			return id
		}
	}
	return ""
}

func hasRequiredRoles(userID string, roles []string) bool {
	// 获取全局授权管理器
	manager := GetGlobalAuthzManager()
	if manager == nil {
		// 如果授权管理器未初始化，记录警告并拒绝访问
		log.Printf("Warning: AuthzManager not initialized, denying access for user %s", userID)
		return false
	}
	
	// 调用真实的角色检查逻辑
	return manager.HasRequiredRoles(userID, roles)
}

func hasRequiredPermissions(userID string, permissions []string) bool {
	// 获取全局授权管理器
	manager := GetGlobalAuthzManager()
	if manager == nil {
		// 如果授权管理器未初始化，记录警告并拒绝访问
		log.Printf("Warning: AuthzManager not initialized, denying access for user %s", userID)
		return false
	}
	
	// 调用真实的权限检查逻辑
	return manager.HasRequiredPermissions(userID, permissions)
}

func evaluateExpression(userID string, expression string) bool {
	// 获取全局授权管理器
	manager := GetGlobalAuthzManager()
	if manager == nil {
		// 如果授权管理器未初始化，记录警告并拒绝访问
		log.Printf("Warning: AuthzManager not initialized, denying access for user %s", userID)
		return false
	}
	
	// 调用真实的表达式评估逻辑
	return manager.EvaluateExpression(userID, expression)
}
`

	tmpl, err := template.New("authz").Funcs(template.FuncMap{
		"sub": func(a, b int) int {
			return a - b
		},
	}).Parse(authzTemplate)
	if err != nil {
		return err
	}

	return tmpl.Execute(file, data)
}

// extractPackageName 提取包名
func (p *AuthzProcessor) extractPackageName(path string) string {
	// 使用共享的包名生成器
	return AuthzPackageGenerator.GetPackageName(path)
}
