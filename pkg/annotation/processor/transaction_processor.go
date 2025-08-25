// Package processor 提供注解处理器实现，用于生成代码
package processor

import (
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/isBlue-5/grain/pkg/annotation/types"
)

// TransactionProcessor 事务注解处理器
type TransactionProcessor struct {
	parser     AnnotationParser
	generator  *Generator
	tmpl       *template.Template
	outputPath string
}

// 事务包装模板
const transactionTmpl = `
// 由事务注解处理器自动生成
package {{.Package}}

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/isBlue-5/grain/pkg/data"
)

{{if .IsMethod}}
// {{.WrappedName}} 是{{.OriginalName}}的事务包装方法
func ({{.ReceiverName}} {{.ReceiverType}}) {{.WrappedName}}({{.Params}}) ({{.Returns}}) {
	{{if .HasTimeout}}
	ctx, cancel := context.WithTimeout({{.ContextName}}, "{{.Timeout}}")
	defer cancel()
	{{else}}
	ctx := {{.ContextName}}
	{{end}}

	// 设置事务选项
	txOpts := &sql.TxOptions{
		{{if .ReadOnly}}
		ReadOnly: true,
		{{end}}
		{{if ne .Isolation "DEFAULT"}}
		Isolation: sql.{{.Isolation}},
		{{end}}
	}

	// 检查上下文中是否已有事务，遵循传播行为
	session := {{.ReceiverName}}.{{.SessionField}}
	{{if eq .Propagation "REQUIRES_NEW"}}
	// REQUIRES_NEW: 始终创建新事务
	tx, err := session.BeginTx(ctx, txOpts)
	if err != nil {
		{{.ErrorReturnFmt}}
	}
	txCtx := data.WithTransaction(ctx, tx)
	
	// 执行原始方法
	{{if .HasReturn}}result, err := {{else}}err := {{end}}{{.ReceiverName}}.{{.OriginalName}}(txCtx, {{.CallArgs}})
	
	// 处理事务提交或回滚
	if err != nil {
		// 检查是否属于需要回滚的错误类型
		{{range .RollbackFor}}
		if _, ok := err.({{.}}); ok {
			tx.Rollback()
			{{$.ErrorReturnErr}}
		}
		{{end}}
		// 默认错误处理
		tx.Rollback()
		{{.ErrorReturnErr}}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		{{.ErrorReturnFmtCommit}}
	}

	{{if .HasReturn}}return result, nil{{else}}return nil{{end}}
	{{else if eq .Propagation "SUPPORTS"}}
	// SUPPORTS: 有事务用事务，无事务则无
	var txCtx context.Context
	var tx data.Transaction
	var needCommit bool

	if data.IsTransactionActive(ctx) {
		// 使用现有事务
		txCtx = ctx
	} else {
		// 不创建新事务
		txCtx = ctx
	}
	
	// 执行原始方法
	{{if .HasReturn}}result, err := {{else}}err := {{end}}{{.ReceiverName}}.{{.OriginalName}}(txCtx, {{.CallArgs}})
	
	// 处理错误
	if err != nil {
		{{.ErrorReturnErr}}
	}

	{{if .HasReturn}}return result, nil{{else}}return nil{{end}}
	{{else if eq .Propagation "NOT_SUPPORTED"}}
	// NOT_SUPPORTED: 不使用事务
	// 如果当前有事务，暂时挂起
	txCtx := ctx
	
	// 执行原始方法
	{{if .HasReturn}}result, err := {{else}}err := {{end}}{{.ReceiverName}}.{{.OriginalName}}(txCtx, {{.CallArgs}})
	
	// 处理错误
	if err != nil {
		{{.ErrorReturnErr}}
	}

	{{if .HasReturn}}return result, nil{{else}}return nil{{end}}
	{{else if eq .Propagation "MANDATORY"}}
	// MANDATORY: 必须在事务中运行
	if !data.IsTransactionActive(ctx) {
		{{.ErrorReturnFmtMandatory}}
	}
	txCtx := ctx
	
	// 执行原始方法
	{{if .HasReturn}}result, err := {{else}}err := {{end}}{{.ReceiverName}}.{{.OriginalName}}(txCtx, {{.CallArgs}})
	
	// 处理错误
	if err != nil {
		{{.ErrorReturnErr}}
	}

	{{if .HasReturn}}return result, nil{{else}}return nil{{end}}
	{{else if eq .Propagation "NEVER"}}
	// NEVER: 不能在事务中运行
	if data.IsTransactionActive(ctx) {
		{{.ErrorReturnFmtNever}}
	}
	txCtx := ctx
	
	// 执行原始方法
	{{if .HasReturn}}result, err := {{else}}err := {{end}}{{.ReceiverName}}.{{.OriginalName}}(txCtx, {{.CallArgs}})
	
	// 处理错误
	if err != nil {
		{{.ErrorReturnErr}}
	}

	{{if .HasReturn}}return result, nil{{else}}return nil{{end}}
	{{else if eq .Propagation "NESTED"}}
	// NESTED: 如果当前存在事务，则在嵌套事务中执行；否则，行为与REQUIRED一样
	var txCtx context.Context
	var tx data.Transaction
	var err error
	var isNested bool

	if data.IsTransactionActive(ctx) {
		// 获取当前事务
		currentTx, _ := data.GetTransaction(ctx)
		// 创建嵌套事务
		tx, err = currentTx.BeginTx(ctx, txOpts)
		if err != nil {
			{{.ErrorReturnFmtNested}}
		}
		txCtx = data.WithTransaction(ctx, tx)
		isNested = true
	} else {
		// 创建新事务
		tx, err = session.BeginTx(ctx, txOpts)
		if err != nil {
			{{.ErrorReturnFmt}}
		}
		txCtx = data.WithTransaction(ctx, tx)
	}

	// 设置Panic恢复，确保事务回滚
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()
	
	// 执行原始方法
	{{if .HasReturn}}result, err := {{else}}err := {{end}}{{.ReceiverName}}.{{.OriginalName}}(txCtx, {{.CallArgs}})
	
	// 处理事务提交或回滚
	if err != nil {
		// 检查是否属于需要回滚的错误类型
		{{range .RollbackFor}}
		if _, ok := err.({{.}}); ok {
			tx.Rollback()
			{{$.ErrorReturnErr}}
		}
		{{end}}
		// 默认错误处理
		tx.Rollback()
		{{.ErrorReturnErr}}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		{{.ErrorReturnFmtCommit}}
	}

	{{if .HasReturn}}return result, nil{{else}}return nil{{end}}
	{{else}}
	// REQUIRED(默认): 有事务用事务，无事务则创建
	var txCtx context.Context
	var tx data.Transaction
	var err error
	var needCommit bool

	if data.IsTransactionActive(ctx) {
		// 使用现有事务
		txCtx = ctx
	} else {
		// 创建新事务
		tx, err = session.BeginTx(ctx, txOpts)
		if err != nil {
			{{.ErrorReturnFmt}}
		}
		txCtx = data.WithTransaction(ctx, tx)
		needCommit = true
		
		// 设置Panic恢复，确保事务回滚
		defer func() {
			if p := recover(); p != nil {
				tx.Rollback()
				panic(p)
			}
		}()
	}
	
	// 执行原始方法
	{{if .HasReturn}}result, err := {{else}}err := {{end}}{{.ReceiverName}}.{{.OriginalName}}(txCtx, {{.CallArgs}})
	
	// 仅当我们创建了新事务时才需要提交或回滚
	if needCommit {
		if err != nil {
			// 检查是否属于需要回滚的错误类型
			{{range .RollbackFor}}
			if _, ok := err.({{.}}); ok {
				tx.Rollback()
				{{$.ErrorReturnErr}}
		}
		{{end}}
		// 默认错误处理
		tx.Rollback()
		{{.ErrorReturnErr}}
	}

		// 提交事务
		if err := tx.Commit(); err != nil {
			{{.ErrorReturnFmtCommit}}
		}
	}

	{{if .HasReturn}}return result, err{{else}}return err{{end}}
	{{end}}
}
{{else}}
// {{.WrappedName}} 是{{.OriginalName}}的事务包装函数
func {{.WrappedName}}({{.Params}}) ({{.Returns}}) {
	// 函数级事务包装实现（与方法类似，但无接收者）
	ctx := context.Background()
	
	// 获取数据库会话
	session := getDBSession() // 需要根据实际项目配置获取
	
	// 开始事务
	tx, err := session.BeginTx(ctx)
	if err != nil {
		return {{range $i, $ret := .Returns}}{{if $i}}, {{end}}{{if eq $ret "error"}}err{{else}}nil{{end}}{{end}}
	}
	
	// 将事务放入上下文
	txCtx := context.WithValue(ctx, "transaction", tx)
	
	// 执行原函数
	result := {{.OriginalName}}({{range $i, $param := .Params}}{{if $i}}, {{end}}{{$param}}{{end}})
	
	// 检查是否有错误
	if {{range $i, $ret := .Returns}}{{if eq $ret "error"}}{{if $i}}, {{end}}result{{else}}{{end}}{{end}} != nil {
		// 回滚事务
		tx.Rollback()
		return {{range $i, $ret := .Returns}}{{if $i}}, {{end}}{{if eq $ret "error"}}result{{else}}nil{{end}}{{end}}
	}
	
	// 提交事务
	if err := tx.Commit(); err != nil {
		return {{range $i, $ret := .Returns}}{{if $i}}, {{end}}{{if eq $ret "error"}}err{{else}}nil{{end}}{{end}}
	}
	
	return {{range $i, $ret := .Returns}}{{if $i}}, {{end}}result{{end}}
}
{{end}}
`

// NewTransactionProcessor 创建事务注解处理器
func NewTransactionProcessor(parser AnnotationParser, generator *Generator, outputPath string) *TransactionProcessor {
	tmpl, err := template.New("transaction").Parse(transactionTmpl)
	if err != nil {
		panic(fmt.Errorf("解析事务模板失败: %w", err))
	}

	return &TransactionProcessor{
		parser:     parser,
		generator:  generator,
		tmpl:       tmpl,
		outputPath: outputPath,
	}
}

// ProcessTransaction 处理事务注解
func (p *TransactionProcessor) ProcessTransaction(paths []string) error {
	// 获取所有事务注解
	var annotations []types.Annotation

	// 解析所有路径下的注解
	for _, path := range paths {
		anns, err := p.parser.ParseDir(path)
		if err != nil {
			return fmt.Errorf("解析目录 %s 失败: %w", path, err)
		}

		for _, ann := range anns {
			if ann.GetType() == types.TransactionType {
				annotations = append(annotations, ann)
			}
		}
	}

	// 按包进行分组处理
	packageMap := make(map[string][]types.CommentAnnotation)
	for _, ann := range annotations {
		if commentAnn, ok := ann.(*types.CommentAnnotation); ok {
			pkgName := filepath.Base(filepath.Dir(commentAnn.Position.Filename))
			packageMap[pkgName] = append(packageMap[pkgName], *commentAnn)
		}
	}

	// 处理每个包的事务注解
	for pkgName, anns := range packageMap {
		if err := p.processPackage(pkgName, anns); err != nil {
			return err
		}
	}

	return nil
}

// processPackage 处理单个包的事务注解
func (p *TransactionProcessor) processPackage(pkgName string, annotations []types.CommentAnnotation) error {
	// 包级别事务处理逻辑
	// 统计包级注解数量和生成包级配置
	packageLevelCount := 0
	for _, ann := range annotations {
		if ann.TargetType == types.PackageTarget {
			packageLevelCount++
		}
	}

	// 如果有包级注解，生成包级配置文件
	if packageLevelCount > 0 {
		// 将CommentAnnotation转换为Annotation接口
		var interfaceAnnotations []types.Annotation
		for i := range annotations {
			interfaceAnnotations = append(interfaceAnnotations, &annotations[i])
		}

		if err := p.generatePackageLevelConfig(pkgName, packageLevelCount, interfaceAnnotations); err != nil {
			fmt.Printf("警告: 生成包级事务配置失败: %v\n", err)
		} else {
			fmt.Printf("包 %s 中发现 %d 个包级事务注解，已生成配置文件\n", pkgName, packageLevelCount)
		}
	}
	for _, ann := range annotations {
		if ann.TargetType == types.MethodTarget {
			if err := p.processMethodAnnotation(pkgName, ann); err != nil {
				return err
			}
		}
	}
	return nil
}

// extractMethodSignature 从方法声明中提取方法签名信息
func (p *TransactionProcessor) extractMethodSignature(methodDecl *ast.FuncDecl) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// 提取接收者信息（如果有）
	if methodDecl.Recv != nil && len(methodDecl.Recv.List) > 0 {
		recvField := methodDecl.Recv.List[0]
		recvType, err := p.extractType(recvField.Type)
		if err != nil {
			return nil, fmt.Errorf("提取接收者类型失败: %w", err)
		}

		// 确定接收者名称
		var recvName string
		if len(recvField.Names) > 0 {
			recvName = recvField.Names[0].Name
		} else {
			recvName = "r" // 默认接收者名称
		}

		result["ReceiverType"] = recvType
		result["ReceiverName"] = recvName
		result["IsMethod"] = true
	} else {
		result["IsMethod"] = false
	}

	// 提取参数信息
	params, ctxParamName, err := p.extractParams(methodDecl.Type.Params)
	if err != nil {
		return nil, fmt.Errorf("提取参数失败: %w", err)
	}
	result["Params"] = params
	result["ContextName"] = ctxParamName

	// 如果没有上下文参数，使用默认值
	if ctxParamName == "" {
		result["ContextName"] = "context.Background()"
	}

	// 提取返回值信息
	returns, hasReturn, hasError, err := p.extractReturns(methodDecl.Type.Results)
	if err != nil {
		return nil, fmt.Errorf("提取返回值失败: %w", err)
	}
	result["Returns"] = returns
	result["HasReturn"] = hasReturn
	result["HasError"] = hasError

	// 生成调用参数列表
	callArgs := p.generateCallArgs(methodDecl.Type.Params, ctxParamName)
	result["CallArgs"] = callArgs

	// 生成错误返回语句
	errorReturn := p.generateErrorReturn(methodDecl.Type.Results)
	result["ErrorReturn"] = errorReturn

	// 生成错误返回语句的辅助函数
	if hasReturn && hasError {
		result["ErrorReturnErr"] = "return nil, err"
		result["ErrorReturnFmt"] = "return nil, fmt.Errorf(\"无法开启事务: %w\", err)"
		result["ErrorReturnFmtCommit"] = "return nil, fmt.Errorf(\"事务提交失败: %w\", err)"
		result["ErrorReturnFmtNested"] = "return nil, fmt.Errorf(\"无法开启嵌套事务: %w\", err)"
		result["ErrorReturnFmtMandatory"] = "return nil, fmt.Errorf(\"MANDATORY传播行为要求已存在事务\")"
		result["ErrorReturnFmtNever"] = "return nil, fmt.Errorf(\"NEVER传播行为不允许在事务中执行\")"
	} else if !hasReturn && hasError {
		result["ErrorReturnErr"] = "return err"
		result["ErrorReturnFmt"] = "return fmt.Errorf(\"无法开启事务: %w\", err)"
		result["ErrorReturnFmtCommit"] = "return fmt.Errorf(\"事务提交失败: %w\", err)"
		result["ErrorReturnFmtNested"] = "return nil, fmt.Errorf(\"无法开启嵌套事务: %w\", err)"
		result["ErrorReturnFmtMandatory"] = "return nil, fmt.Errorf(\"MANDATORY传播行为要求已存在事务\")"
		result["ErrorReturnFmtNever"] = "return nil, fmt.Errorf(\"NEVER传播行为不允许在事务中执行\")"
	} else if hasReturn && !hasError {
		result["ErrorReturnErr"] = "return nil"
		result["ErrorReturnFmt"] = "return nil"
		result["ErrorReturnFmtCommit"] = "return nil"
		result["ErrorReturnFmtNested"] = "return nil"
		result["ErrorReturnFmtMandatory"] = "return nil"
		result["ErrorReturnFmtNever"] = "return nil"
	} else {
		result["ErrorReturnErr"] = "return"
		result["ErrorReturnFmt"] = "return"
		result["ErrorReturnFmtCommit"] = "return"
		result["ErrorReturnFmtNested"] = "return"
		result["ErrorReturnFmtMandatory"] = "return"
		result["ErrorReturnFmtNever"] = "return"
	}

	return result, nil
}

// extractType 提取类型表达式的字符串表示
func (p *TransactionProcessor) extractType(expr ast.Expr) (string, error) {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name, nil
	case *ast.StarExpr:
		baseType, err := p.extractType(t.X)
		if err != nil {
			return "", err
		}
		return "*" + baseType, nil
	case *ast.SelectorExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name + "." + t.Sel.Name, nil
		}
		return "", fmt.Errorf("不支持的选择器表达式")
	case *ast.ArrayType:
		elemType, err := p.extractType(t.Elt)
		if err != nil {
			return "", err
		}
		return "[]" + elemType, nil
	case *ast.MapType:
		keyType, err := p.extractType(t.Key)
		if err != nil {
			return "", err
		}
		valueType, err := p.extractType(t.Value)
		if err != nil {
			return "", err
		}
		return "map[" + keyType + "]" + valueType, nil
	case *ast.InterfaceType:
		return "interface{}", nil
	case *ast.FuncType:
		return "func()", nil // 简化函数类型表示
	case *ast.ChanType:
		elemType, err := p.extractType(t.Value)
		if err != nil {
			return "", err
		}
		if t.Dir == ast.SEND {
			return "chan<- " + elemType, nil
		} else if t.Dir == ast.RECV {
			return "<-chan " + elemType, nil
		}
		return "chan " + elemType, nil
	case *ast.StructType:
		return "struct{}", nil // 简化结构体类型表示
	case *ast.Ellipsis:
		elemType, err := p.extractType(t.Elt)
		if err != nil {
			return "", err
		}
		return "..." + elemType, nil
	default:
		return "", fmt.Errorf("不支持的类型表达式: %T", expr)
	}
}

// extractParams 提取函数参数信息
func (p *TransactionProcessor) extractParams(fields *ast.FieldList) (string, string, error) {
	if fields == nil || len(fields.List) == 0 {
		return "", "", nil
	}

	var params []string
	var ctxParamName string

	for _, field := range fields.List {
		fieldType, err := p.extractType(field.Type)
		if err != nil {
			return "", "", err
		}

		// 检查是否为上下文参数
		isCtx := false
		if sel, ok := field.Type.(*ast.SelectorExpr); ok {
			if x, ok := sel.X.(*ast.Ident); ok && x.Name == "context" && sel.Sel.Name == "Context" {
				isCtx = true
			}
		}

		if len(field.Names) == 0 {
			// 匿名参数
			params = append(params, fieldType)
			if isCtx {
				ctxParamName = "_ctx" // 为匿名上下文参数指定默认名称
			}
		} else {
			for _, name := range field.Names {
				params = append(params, name.Name+" "+fieldType)
				if isCtx {
					ctxParamName = name.Name
				}
			}
		}
	}

	return strings.Join(params, ", "), ctxParamName, nil
}

// extractReturns 提取函数返回值信息
func (p *TransactionProcessor) extractReturns(fields *ast.FieldList) (string, bool, bool, error) {
	if fields == nil || len(fields.List) == 0 {
		return "", false, false, nil
	}

	var returns []string
	hasReturn := false
	hasError := false

	for _, field := range fields.List {
		fieldType, err := p.extractType(field.Type)
		if err != nil {
			return "", false, false, err
		}

		// 检查是否为error类型
		if ident, ok := field.Type.(*ast.Ident); ok && ident.Name == "error" {
			hasError = true
		}

		// 除了error类型外，其他都视为返回值
		if fieldType != "error" {
			hasReturn = true
		}

		if len(field.Names) == 0 {
			// 匿名返回值
			returns = append(returns, fieldType)
		} else {
			for _, name := range field.Names {
				returns = append(returns, name.Name+" "+fieldType)
			}
		}
	}

	return strings.Join(returns, ", "), hasReturn, hasError, nil
}

// generateCallArgs 生成调用参数列表
func (p *TransactionProcessor) generateCallArgs(fields *ast.FieldList, ctxParamName string) string {
	if fields == nil || len(fields.List) == 0 {
		return ""
	}

	var args []string
	for _, field := range fields.List {
		// 跳过上下文参数，因为我们会传递事务上下文
		isCtx := false
		if sel, ok := field.Type.(*ast.SelectorExpr); ok {
			if x, ok := sel.X.(*ast.Ident); ok && x.Name == "context" && sel.Sel.Name == "Context" {
				isCtx = true
			}
		}

		if len(field.Names) == 0 {
			// 匿名参数，使用类型名作为参数名
			if !isCtx {
				typeName, _ := p.extractType(field.Type)
				// 将首字母小写作为参数名
				paramName := strings.ToLower(typeName[:1])
				args = append(args, paramName)
			}
		} else {
			for _, name := range field.Names {
				if !isCtx || name.Name != ctxParamName {
					args = append(args, name.Name)
				}
			}
		}
	}

	return strings.Join(args, ", ")
}

// generateErrorReturn 生成错误返回语句
func (p *TransactionProcessor) generateErrorReturn(fields *ast.FieldList) string {
	if fields == nil || len(fields.List) == 0 {
		return "return"
	}

	var returns []string
	for _, field := range fields.List {
		fieldType, _ := p.extractType(field.Type)

		// 根据类型生成默认返回值
		var defaultValue string
		if ident, ok := field.Type.(*ast.Ident); ok && ident.Name == "error" {
			defaultValue = "err"
		} else {
			switch fieldType {
			case "string":
				defaultValue = `""`
			case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64":
				defaultValue = "0"
			case "bool":
				defaultValue = "false"
			default:
				if strings.HasPrefix(fieldType, "*") || strings.HasPrefix(fieldType, "[]") || strings.HasPrefix(fieldType, "map[") {
					defaultValue = "nil"
				} else {
					defaultValue = fieldType + "{}"
				}
			}
		}

		returns = append(returns, defaultValue)
	}

	return "return " + strings.Join(returns, ", ")
}

// processMethodAnnotation 处理方法上的事务注解
func (p *TransactionProcessor) processMethodAnnotation(pkgName string, ann types.CommentAnnotation) error {
	// 提取注解参数
	timeout := types.GetStringAttribute(ann.Attributes, "timeout", "5s")
	hasTimeout := timeout != ""

	propagation := types.GetStringAttribute(ann.Attributes, "propagation", "REQUIRED")

	// 解析回滚类型
	rollbackFor, _ := types.GetStringSliceAttribute(ann.Attributes, "rollbackFor")
	if len(rollbackFor) == 0 {
		rollbackFor = []string{"error"} // 默认对所有error进行回滚
	}

	// 解析隔离级别
	isolation := types.GetStringAttribute(ann.Attributes, "isolation", "DEFAULT")

	// 解析只读标志
	readOnly := types.GetBoolAttribute(ann.Attributes, "readOnly", false)

	// 分析方法
	methodDecl, ok := ann.Node.(*ast.FuncDecl)
	if !ok {
		return fmt.Errorf("事务注解只能应用于方法: %s", ann.TargetName)
	}

	// 提取方法签名信息
	signatureInfo, err := p.extractMethodSignature(methodDecl)
	if err != nil {
		return fmt.Errorf("提取方法签名失败: %w", err)
	}

	// 获取原始方法名和包装方法名
	originalName := methodDecl.Name.Name
	wrappedName := originalName
	originalName = originalName + "Impl" // 原方法将被重命名

	// 将方法签名信息添加到模板数据
	tmplData := map[string]interface{}{
		"Package":      pkgName,
		"WrappedName":  wrappedName,
		"OriginalName": originalName,
		"HasTimeout":   hasTimeout,
		"Timeout":      timeout,
		"Propagation":  propagation,
		"RollbackFor":  rollbackFor,
		"Isolation":    isolation,
		"ReadOnly":     readOnly,
		"SessionField": getSessionFieldName(methodDecl), // 智能获取会话字段名
	}

	// 合并方法签名信息
	for k, v := range signatureInfo {
		tmplData[k] = v
	}

	// 生成包装代码文件
	outputFile := filepath.Join(p.outputPath, fmt.Sprintf("%s_tx.go", strings.ToLower(ann.TargetName)))

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

	// 执行模板
	if err := p.tmpl.Execute(file, tmplData); err != nil {
		return fmt.Errorf("生成事务包装代码失败: %w", err)
	}

	return nil
}

// getSessionFieldName 智能获取会话字段名
func getSessionFieldName(methodDecl *ast.FuncDecl) string {
	// 如果是方法，检查接收者类型的字段
	if methodDecl.Recv != nil && len(methodDecl.Recv.List) > 0 {
		// 通过类型分析找到实际的字段名
		recvType := methodDecl.Recv.List[0].Type

		// 尝试从接收者类型推断字段名
		if starExpr, ok := recvType.(*ast.StarExpr); ok {
			// 指针类型，如 *UserService
			if ident, ok := starExpr.X.(*ast.Ident); ok {
				// 根据类型名推断字段名
				typeName := strings.ToLower(ident.Name)
				if strings.Contains(typeName, "service") {
					return "db" // 服务层通常使用db
				} else if strings.Contains(typeName, "repository") {
					return "db" // 仓库层通常使用db
				} else if strings.Contains(typeName, "controller") {
					return "db" // 控制器层通常使用db
				} else if strings.Contains(typeName, "handler") {
					return "db" // 处理器层通常使用db
				}
			}
		} else if ident, ok := recvType.(*ast.Ident); ok {
			// 值类型，如 UserService
			typeName := strings.ToLower(ident.Name)
			if strings.Contains(typeName, "service") {
				return "db"
			} else if strings.Contains(typeName, "repository") {
				return "db"
			} else if strings.Contains(typeName, "controller") {
				return "db"
			} else if strings.Contains(typeName, "handler") {
				return "db"
			}
		}

		// 如果无法推断，使用最常见的名称
		return "db"
	}

	// 默认返回最常见的字段名
	return "db"
}

// generatePackageLevelConfig 生成包级事务配置
func (p *TransactionProcessor) generatePackageLevelConfig(pkgName string, count int, annotations []types.Annotation) error {
	// 创建包级配置文件
	configData := map[string]interface{}{
		"PackageName":        pkgName,
		"AnnotationCount":    count,
		"DefaultTimeout":     "30s",
		"DefaultIsolation":   "READ_COMMITTED",
		"DefaultPropagation": "REQUIRED",
	}

	// 分析注解，提取配置信息
	for _, ann := range annotations {
		if ann.GetType() == "transaction" {
			if commentAnn, ok := ann.(*types.CommentAnnotation); ok {
				// 提取超时配置
				if timeout := p.extractTimeout(commentAnn); timeout != "" {
					configData["DefaultTimeout"] = timeout
				}
				// 提取隔离级别
				if isolation := p.extractIsolation(commentAnn); isolation != "" {
					configData["DefaultIsolation"] = isolation
				}
				// 提取传播行为
				if propagation := p.extractPropagation(commentAnn); propagation != "" {
					configData["DefaultPropagation"] = propagation
				}
			}
		}
	}

	// 生成配置文件内容
	configContent := fmt.Sprintf(`// 包级事务配置
// 自动生成，请勿手动修改
package %s

import (
	"time"
	"github.com/isBlue-5/grain/pkg/data/transaction"
)

// TransactionConfig 事务配置
var TransactionConfig = &transaction.Config{
	DefaultTimeout:     %s,
	DefaultIsolation:   transaction.%s,
	DefaultPropagation: transaction.%s,
	MaxRetries:         3,
	RetryDelay:         time.Second,
}

// GetTransactionConfig 获取事务配置
func GetTransactionConfig() *transaction.Config {
	return TransactionConfig
}
`, pkgName, configData["DefaultTimeout"], configData["DefaultIsolation"], configData["DefaultPropagation"])

	// 写入配置文件
	configFile := filepath.Join(p.outputPath, pkgName, "transaction_config.go")
	if err := os.MkdirAll(filepath.Dir(configFile), 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// extractTimeout 提取超时配置
func (p *TransactionProcessor) extractTimeout(ann *types.CommentAnnotation) string {
	// 从注解内容中提取超时配置
	content := ann.Raw
	if strings.Contains(content, "timeout:") {
		parts := strings.Split(content, "timeout:")
		if len(parts) > 1 {
			timeout := strings.TrimSpace(strings.Split(parts[1], " ")[0])
			return timeout
		}
	}
	return ""
}

// extractIsolation 提取隔离级别
func (p *TransactionProcessor) extractIsolation(ann *types.CommentAnnotation) string {
	// 从注解内容中提取隔离级别
	content := ann.Raw
	if strings.Contains(content, "isolation:") {
		parts := strings.Split(content, "isolation:")
		if len(parts) > 1 {
			isolation := strings.TrimSpace(strings.Split(parts[1], " ")[0])
			return strings.ToUpper(isolation)
		}
	}
	return "READ_COMMITTED"
}

// extractPropagation 提取传播行为
func (p *TransactionProcessor) extractPropagation(ann *types.CommentAnnotation) string {
	// 从注解内容中提取传播行为
	content := ann.Raw
	if strings.Contains(content, "propagation:") {
		parts := strings.Split(content, "propagation:")
		if len(parts) > 1 {
			propagation := strings.TrimSpace(strings.Split(parts[1], " ")[0])
			return strings.ToUpper(propagation)
		}
	}
	return "REQUIRED"
}
