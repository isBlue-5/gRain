// Package processor 提供注解处理器的实现
package processor

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/isBlue-5/grain/pkg/annotation/types"
)

// AnnotationParser 注解解析器接口
type AnnotationParser interface {
	// ParseFile 解析单个文件中的注解
	ParseFile(filename string) ([]types.Annotation, error)

	// ParsePackage 解析整个包中的注解
	ParsePackage(pkgPath string) ([]types.Annotation, error)

	// ParseDir 解析目录中的注解
	ParseDir(dirPath string) ([]types.Annotation, error)
}

// DefaultAnnotationParser 默认注解解析器实现
type DefaultAnnotationParser struct {
	// 注解前缀，例如 "frame:"
	prefix string

	// 注释注解的正则表达式
	commentPattern *regexp.Regexp

	// 文件集合，用于跟踪源代码位置
	fileSet *token.FileSet
}

// NewAnnotationParser 创建新的注解解析器
func NewAnnotationParser(prefix string) *DefaultAnnotationParser {
	// 创建解析注释注解的正则表达式
	// 例如: // frame:route(method="GET", path="/users")
	pattern := fmt.Sprintf(`//\s*%s(\w+)\s*(?:\((.*?)\))?`, prefix)

	return &DefaultAnnotationParser{
		prefix:         prefix,
		commentPattern: regexp.MustCompile(pattern),
		fileSet:        token.NewFileSet(),
	}
}

// ParseFile 解析单个文件中的注解
func (p *DefaultAnnotationParser) ParseFile(filename string) ([]types.Annotation, error) {
	// 解析源文件
	file, err := parser.ParseFile(p.fileSet, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("解析文件失败: %w", err)
	}

	// 解析文件中的所有注解
	return p.parseAST(file), nil
}

// ParsePackage 解析整个包中的注解
func (p *DefaultAnnotationParser) ParsePackage(pkgPath string) ([]types.Annotation, error) {
	// 获取绝对路径
	absPath, err := filepath.Abs(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("获取绝对路径失败: %w", err)
	}

	fmt.Printf("解析包路径: %s (绝对路径: %s)\n", pkgPath, absPath)

	var allAnnotations []types.Annotation

	// 递归遍历目录，找到所有Go文件
	err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只处理Go文件，并过滤生成的文件
		if !info.IsDir() && strings.HasSuffix(path, ".go") && p.shouldParseFile(path) {
			fmt.Printf("解析文件: %s\n", path)
			file, err := parser.ParseFile(p.fileSet, path, nil, parser.ParseComments)
			if err != nil {
				fmt.Printf("解析文件 %s 失败: %v\n", path, err)
				return nil // 继续处理其他文件
			}

			annotations := p.parseAST(file)
			fmt.Printf("从文件 %s 解析到 %d 个注解\n", path, len(annotations))
			allAnnotations = append(allAnnotations, annotations...)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("遍历目录失败: %w", err)
	}

	return allAnnotations, nil
}

// ParseDir 解析目录中的注解
func (p *DefaultAnnotationParser) ParseDir(dirPath string) ([]types.Annotation, error) {
	// 直接调用解析包的方法
	return p.ParsePackage(dirPath)
}

// parseAST 从AST中解析注解
func (p *DefaultAnnotationParser) parseAST(file *ast.File) []types.Annotation {
	var annotations []types.Annotation

	// 跟踪包信息用于依赖分析
	pkgName := file.Name.Name
	fileName := p.fileSet.Position(file.Pos()).Filename

	// 构建完整上下文，提供更丰富的解析环境
	ctx := &parserContext{
		pkgName:  pkgName,
		fileName: fileName,
		imports:  extractImports(file),
		typeInfo: make(map[string]*typeInfo),
	}

	// 第一遍：收集所有类型定义
	ast.Inspect(file, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.TypeSpec:
			// 记录类型信息
			if n.Name != nil {
				info := &typeInfo{
					name:     n.Name.Name,
					typeSpec: n,
				}

				// 检查是否为结构体
				if structType, ok := n.Type.(*ast.StructType); ok {
					info.isStruct = true
					info.structType = structType
				}

				ctx.typeInfo[n.Name.Name] = info
			}
		}
		return true
	})

	// 第二遍：处理注解
	ast.Inspect(file, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.StructType:
			// 处理结构体类型
			if n.Fields != nil {
				// 确定结构体名称
				structName := getStructNameFromNode(n, ctx)

				// 检查每个字段的标签
				for _, field := range n.Fields.List {
					if field.Tag != nil {
						// 提取标签值
						tagValue := field.Tag.Value

						// 去除首尾的引号
						tagValue = strings.Trim(tagValue, "`")

						// 寻找注解标签
						tagAnnotations := p.parseStructTag(tagValue, field, n)

						// 如果找到结构体名称，更新注解信息
						if structName != "" {
							for i := range tagAnnotations {
								if tagAnno, ok := tagAnnotations[i].(*types.StructTagAnnotation); ok {
									tagAnno.StructName = structName
								}
							}
						}

						annotations = append(annotations, tagAnnotations...)
					}
				}
			}

		case *ast.GenDecl:
			// 处理声明（可能包含结构体定义）
			if n.Doc != nil {
				// 处理声明前的注释
				for _, comment := range n.Doc.List {
					if ann := p.parseComment(comment, n, ctx); ann != nil {
						// 对于类型声明，确保Node字段指向正确的TypeSpec
						if n.Tok == token.TYPE && len(n.Specs) > 0 {
							if typeSpec, ok := n.Specs[0].(*ast.TypeSpec); ok {
								// 更新注解的Node字段为TypeSpec
								ann.Node = typeSpec
							}
						}
						annotations = append(annotations, ann)
					}
				}
			}

		case *ast.FuncDecl:
			// 处理函数和方法声明
			if n.Doc != nil {
				// 增强方法处理，确定接收者
				methodCtx := ctx.clone()
				if n.Recv != nil && len(n.Recv.List) > 0 {
					methodCtx.receiverType = extractReceiverType(n.Recv.List[0].Type)
				}

				// 处理函数/方法前的注释
				for _, comment := range n.Doc.List {
					if ann := p.parseComment(comment, n, methodCtx); ann != nil {
						annotations = append(annotations, ann)
					}
				}
			}
		}

		return true
	})

	// 处理文件级别的注释
	if file.Doc != nil {
		for _, comment := range file.Doc.List {
			if ann := p.parseComment(comment, file, ctx); ann != nil {
				annotations = append(annotations, ann)
			}
		}
	}

	return annotations
}

// 解析上下文，用于提供更完整的解析环境
type parserContext struct {
	pkgName      string
	fileName     string
	imports      map[string]string
	receiverType string
	typeInfo     map[string]*typeInfo
}

// 克隆上下文
func (c *parserContext) clone() *parserContext {
	return &parserContext{
		pkgName:      c.pkgName,
		fileName:     c.fileName,
		imports:      c.imports,
		receiverType: c.receiverType,
		typeInfo:     c.typeInfo,
	}
}

// 类型信息
type typeInfo struct {
	name       string
	typeSpec   *ast.TypeSpec
	isStruct   bool
	structType *ast.StructType
}

// extractImports 提取导入信息
func extractImports(file *ast.File) map[string]string {
	imports := make(map[string]string)

	for _, imp := range file.Imports {
		// 去除引号
		path := strings.Trim(imp.Path.Value, "\"")

		// 处理命名导入
		if imp.Name != nil {
			imports[imp.Name.Name] = path
		} else {
			// 获取包名作为默认名称
			parts := strings.Split(path, "/")
			name := parts[len(parts)-1]
			imports[name] = path
		}
	}

	return imports
}

// getStructNameFromNode 尝试从节点获取结构体名称
func getStructNameFromNode(structType *ast.StructType, ctx *parserContext) string {
	// 尝试反向查找结构体名称
	for name, info := range ctx.typeInfo {
		if info.isStruct && info.structType == structType {
			return name
		}
	}

	return ""
}

// parseStructTag 解析结构体标签
func (p *DefaultAnnotationParser) parseStructTag(tag string, field *ast.Field, structType *ast.StructType) []types.Annotation {
	var annotations []types.Annotation

	// 解析标签
	tags := splitTags(tag)

	// 检查每个标签是否是注解
	for tagName, tagValue := range tags {
		// 检查是否以指定前缀开始
		if strings.HasPrefix(tagName, p.prefix) {
			// 提取注解类型
			annotationType := strings.TrimPrefix(tagName, p.prefix)

			// 创建注解
			annotation := &types.StructTagAnnotation{
				Type:        types.AnnotationType(annotationType),
				TargetType:  types.FieldTarget,
				TargetName:  getFieldName(field),
				TagName:     tagName,
				TagValue:    tagValue,
				Position:    types.SourcePosition(p.fileSet, field.Pos()),
				StructField: field,
				StructType:  structType,
			}

			// 解析属性
			annotation.Attributes = parseComplexAttributes(tagValue)

			annotations = append(annotations, annotation)
		}
	}

	return annotations
}

// extractReceiverType 提取接收者类型名称
func extractReceiverType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		// 指针类型，如 (*Controller)
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		} else if sel, ok := t.X.(*ast.SelectorExpr); ok {
			// 处理导入的类型，如 (*pkg.Type)
			return sel.Sel.Name
		}
	case *ast.Ident:
		// 值类型，如 (Controller)
		return t.Name
	case *ast.SelectorExpr:
		// 导入的类型，如 (pkg.Type)
		return t.Sel.Name
	}

	return ""
}

// parseComment 解析注释中的注解
func (p *DefaultAnnotationParser) parseComment(comment *ast.Comment, node ast.Node, ctx *parserContext) *types.CommentAnnotation {
	// 检查注释是否包含注解
	matches := p.commentPattern.FindStringSubmatch(comment.Text)
	if len(matches) < 2 {
		return nil
	}

	// 提取注解类型和属性
	annotationType := matches[1]
	attributes := ""
	if len(matches) > 2 {
		attributes = matches[2]
	}

	// 添加调试信息
	fmt.Printf("DEBUG: 解析到注解: %s, 类型: %s, 属性: %s\n", comment.Text, annotationType, attributes)

	// 确定目标类型和名称
	targetType, targetName := p.determineTarget(node, ctx)

	// 创建注解
	annotation := &types.CommentAnnotation{
		Type:       types.AnnotationType(annotationType),
		TargetType: targetType,
		TargetName: targetName,
		Raw:        comment.Text,
		Position:   types.SourcePosition(p.fileSet, comment.Pos()),
		Comment:    comment,
		Node:       node,
	}

	// 使用更强大的属性解析
	annotation.Attributes = parseComplexAttributes(attributes)

	// 对于方法注解，添加接收者信息
	if ctx.receiverType != "" && targetType == types.MethodTarget {
		annotation.ReceiverType = ctx.receiverType
	}

	return annotation
}

// determineTarget 确定注解的目标类型和名称
func (p *DefaultAnnotationParser) determineTarget(node ast.Node, ctx *parserContext) (types.TargetType, string) {
	switch n := node.(type) {
	case *ast.File:
		// 文件级注解
		return types.PackageTarget, ctx.pkgName

	case *ast.GenDecl:
		// 一般声明
		if n.Specs != nil && len(n.Specs) > 0 {
			if typeSpec, ok := n.Specs[0].(*ast.TypeSpec); ok {
				// 类型声明
				return types.TypeTarget, typeSpec.Name.Name
			} else if valueSpec, ok := n.Specs[0].(*ast.ValueSpec); ok {
				// 变量或常量声明
				if len(valueSpec.Names) > 0 {
					return types.VariableTarget, valueSpec.Names[0].Name
				}
			}
		}
		return types.TypeTarget, ""

	case *ast.FuncDecl:
		// 函数或方法声明
		if n.Recv != nil {
			// 方法
			return types.MethodTarget, n.Name.Name
		}
		// 函数
		return types.FunctionTarget, n.Name.Name

	default:
		return "", ""
	}
}

// parseComplexAttributes 解析复杂的注解属性，支持嵌套结构
func parseComplexAttributes(attrStr string) []types.AnnotationAttribute {
	if strings.TrimSpace(attrStr) == "" {
		return nil
	}

	var attributes []types.AnnotationAttribute
	var currentKey string
	var currentValue string
	var inQuote bool
	var quoteChar rune
	var depth int

	// 添加结束字符以简化处理
	attrStr = strings.TrimSpace(attrStr) + ","

	for i, r := range attrStr {
		switch {
		case (r == '"' || r == '\'') && (i == 0 || attrStr[i-1] != '\\'):
			// 处理引号
			if !inQuote {
				inQuote = true
				quoteChar = r
			} else if r == quoteChar {
				inQuote = false
			} else {
				currentValue += string(r)
			}

		case r == '{' && !inQuote:
			// 处理对象开始
			depth++
			currentValue += string(r)

		case r == '}' && !inQuote:
			// 处理对象结束
			depth--
			currentValue += string(r)

		case r == ',' && !inQuote && depth == 0:
			// 处理属性分隔符
			pair := strings.SplitN(currentKey+currentValue, "=", 2)
			if len(pair) == 2 {
				key := strings.TrimSpace(pair[0])
				value := strings.TrimSpace(pair[1])

				// 去除引号
				if len(value) >= 2 && (value[0] == '"' && value[len(value)-1] == '"' ||
					value[0] == '\'' && value[len(value)-1] == '\'') {
					value = value[1 : len(value)-1]
				}

				// 尝试解析值
				parsedValue := parseAttributeValue(value)
				attributes = append(attributes, types.AnnotationAttribute{
					Name:  key,
					Value: parsedValue,
				})
			} else if currentKey+currentValue != "" {
				// 无值的属性，例如 frame:route(GET)
				flag := strings.TrimSpace(currentKey + currentValue)
				attributes = append(attributes, types.AnnotationAttribute{
					Name:  flag,
					Value: true,
				})
			}
			currentKey = ""
			currentValue = ""

		case r == '=' && !inQuote && depth == 0:
			// 处理键值分隔符
			currentKey = currentValue
			currentValue = ""

		default:
			// 累积当前值
			currentValue += string(r)
		}
	}

	return attributes
}

// parseAttributeValue 解析属性值，支持基本类型和JSON格式的复杂类型
func parseAttributeValue(value string) interface{} {
	// 尝试解析布尔值
	if value == "true" {
		return true
	} else if value == "false" {
		return false
	}

	// 尝试解析数字
	if i, err := strconv.Atoi(value); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f
	}

	// 尝试解析JSON数组或对象
	if strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}") ||
		strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		var result interface{}
		if err := json.Unmarshal([]byte(value), &result); err == nil {
			return result
		}
	}

	// 默认返回字符串
	return value
}

// splitTags 将结构体标签字符串分割成键值对
func splitTags(tag string) map[string]string {
	tags := make(map[string]string)

	// 使用空格分割标签
	parts := strings.Fields(tag)

	for _, part := range parts {
		// 寻找第一个冒号
		i := strings.IndexByte(part, ':')
		if i == -1 {
			// 没有找到冒号，跳过
			continue
		}

		key := part[:i]
		value := ""

		// 提取值部分（可能包含在引号中）
		if i+1 < len(part) {
			value = part[i+1:]
			value = strings.Trim(value, `"`)
		}

		tags[key] = value
	}

	return tags
}

// parseAttributes 解析注解属性
func parseAttributes(attrStr string) []types.AnnotationAttribute {
	if attrStr == "" {
		return nil
	}

	var attributes []types.AnnotationAttribute

	// 简单解析属性（这里可以改进为更复杂的解析器）
	pairs := strings.Split(attrStr, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		keyValue := strings.SplitN(pair, "=", 2)

		if len(keyValue) == 2 {
			key := strings.TrimSpace(keyValue[0])
			value := strings.TrimSpace(keyValue[1])

			// 移除引号
			value = strings.Trim(value, `"`)

			attributes = append(attributes, types.AnnotationAttribute{
				Name:  key,
				Value: value,
			})
		} else if len(keyValue) == 1 && keyValue[0] != "" {
			// 无值的属性，例如 frame:route(GET)
			attributes = append(attributes, types.AnnotationAttribute{
				Name:  keyValue[0],
				Value: true,
			})
		}
	}

	return attributes
}

// getFieldName 获取字段名称
func getFieldName(field *ast.Field) string {
	if field.Names != nil && len(field.Names) > 0 {
		return field.Names[0].Name
	}

	// 对于匿名字段，尝试获取类型名称
	switch typeExpr := field.Type.(type) {
	case *ast.Ident:
		return typeExpr.Name
	case *ast.SelectorExpr:
		return typeExpr.Sel.Name
	case *ast.StarExpr:
		if ident, ok := typeExpr.X.(*ast.Ident); ok {
			return "*" + ident.Name
		}
	}

	return ""
}

// shouldParseFile 判断是否应该解析指定文件，过滤生成的文件和其他不需要的文件
func (p *DefaultAnnotationParser) shouldParseFile(filePath string) bool {
	fileName := filepath.Base(filePath)

	// 过滤生成的文件
	if strings.HasSuffix(fileName, "_gen.go") ||
		strings.HasSuffix(fileName, "_generated.go") ||
		strings.HasSuffix(fileName, ".pb.go") ||
		strings.HasSuffix(fileName, ".pb.gw.go") {
		return false
	}

	// 过滤测试文件（可选，取决于需求）
	if strings.HasSuffix(fileName, "_test.go") {
		return false
	}

	// 过滤生成目录下的所有文件
	if strings.Contains(filePath, "/generated/") ||
		strings.Contains(filePath, "\\generated\\") {
		return false
	}

	// 过滤常见的工具生成目录
	if strings.Contains(filePath, "/.git/") ||
		strings.Contains(filePath, "/vendor/") ||
		strings.Contains(filePath, "/node_modules/") ||
		strings.Contains(filePath, "/.cache/") {
		return false
	}

	return true
}
