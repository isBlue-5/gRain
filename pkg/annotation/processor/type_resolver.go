package processor

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
)

// TypeResolver 类型解析器
type TypeResolver struct {
	packages  map[string]*ast.Package
	typeCache map[string]*OpenAPISchema
	workDir   string
	// 添加类型注释映射，用于存储类型名称到注释的映射
	typeComments map[string]string
}

// NewTypeResolver 创建新的类型解析器
func NewTypeResolver(workDir string) *TypeResolver {
	return &TypeResolver{
		packages:     make(map[string]*ast.Package),
		typeCache:    make(map[string]*OpenAPISchema),
		workDir:      workDir,
		typeComments: make(map[string]string),
	}
}

// ResolveType 解析Go类型为OpenAPI Schema
func (tr *TypeResolver) ResolveType(typeName string) (*OpenAPISchema, error) {
	// 检查缓存
	if schema, exists := tr.typeCache[typeName]; exists {
		return schema, nil
	}

	// 解析类型名称
	packagePath, structName := tr.parseTypeName(typeName)
	if packagePath == "" || structName == "" {
		return nil, fmt.Errorf("invalid type name: %s", typeName)
	}

	// 查找包
	pkg, err := tr.findPackage(packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to find package %s: %w", packagePath, err)
	}

	// 查找类型定义
	typeSpec, err := tr.findTypeSpec(pkg, structName)
	if err != nil {
		return nil, fmt.Errorf("failed to find type %s: %w", structName, err)
	}

	// 解析类型
	schema := tr.parseTypeSpec(structName, typeSpec)

	// 缓存结果
	tr.typeCache[typeName] = schema

	return schema, nil
}

// findPackage 查找包
func (tr *TypeResolver) findPackage(packagePath string) (*ast.Package, error) {
	if pkg, exists := tr.packages[packagePath]; exists {
		return pkg, nil
	}

	// 构建完整路径
	fullPath := filepath.Join(tr.workDir, packagePath)

	// 解析包
	pkgs, err := parser.ParseDir(token.NewFileSet(), fullPath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// 取第一个包
	for _, pkg := range pkgs {
		tr.packages[packagePath] = pkg
		return pkg, nil
	}

	return nil, fmt.Errorf("no package found in %s", fullPath)
}

// findTypeSpec 查找类型定义
func (tr *TypeResolver) findTypeSpec(pkg *ast.Package, typeName string) (*ast.TypeSpec, error) {
	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			if genDecl, ok := decl.(*ast.GenDecl); ok {
				// 提取注释
				var comment string
				if genDecl.Doc != nil {
					comment = strings.TrimSpace(genDecl.Doc.Text())
				}

				for _, spec := range genDecl.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						if typeSpec.Name.Name == typeName {
							// 存储类型注释
							tr.typeComments[typeName] = comment
							return typeSpec, nil
						}
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("type %s not found", typeName)
}

// parseTypeSpec 解析类型定义
func (tr *TypeResolver) parseTypeSpec(typeName string, typeSpec *ast.TypeSpec) *OpenAPISchema {
	switch t := typeSpec.Type.(type) {
	case *ast.StructType:
		return tr.parseStructType(typeName, t)
	case *ast.ArrayType:
		return tr.parseArrayType(t)
	case *ast.StarExpr:
		return tr.parsePointerType(t)
	case *ast.InterfaceType:
		return tr.parseInterfaceType(typeName, t)
	case *ast.Ident:
		return tr.parseBasicType(t.Name)
	default:
		return &OpenAPISchema{
			Type: "string",
		}
	}
}

// parseStructType 解析结构体类型
func (tr *TypeResolver) parseStructType(structName string, structType *ast.StructType) *OpenAPISchema {
	// 获取结构体注释
	var structComment string
	if comment, exists := tr.typeComments[structName]; exists {
		structComment = comment
	}

	schema := &OpenAPISchema{
		Type:        "object",
		Properties:  make(map[string]*OpenAPISchema),
		Required:    make([]string, 0),
		Description: structComment,
	}

	// 解析字段
	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			continue
		}

		fieldName := field.Names[0].Name
		fieldSchema := tr.parseFieldType(field.Type)

		// 解析标签
		var jsonFieldName string
		if field.Tag != nil {
			jsonFieldName = tr.extractJSONFieldName(field.Tag.Value)
			fieldSchema = tr.parseFieldTags(fieldSchema, field.Tag.Value)
		}

		// 如果没有JSON标签或字段被忽略，跳过此字段
		if jsonFieldName == "" || jsonFieldName == "-" {
			continue
		}

		// 使用JSON字段名作为属性名
		if jsonFieldName == "" {
			jsonFieldName = fieldName
		}

		// 解析字段注释 - 优先使用行注释，然后是块注释
		var fieldComment string
		if field.Comment != nil {
			fieldComment = strings.TrimSpace(field.Comment.Text())
		}
		if field.Doc != nil {
			if fieldComment != "" {
				fieldComment = strings.TrimSpace(field.Doc.Text()) + " " + fieldComment
			} else {
				fieldComment = strings.TrimSpace(field.Doc.Text())
			}
		}

		if fieldComment != "" {
			fieldSchema.Description = fieldComment
		}

		// 检查是否必需
		if tr.isFieldRequired(field) {
			schema.Required = append(schema.Required, jsonFieldName)
		}

		schema.Properties[jsonFieldName] = fieldSchema
	}

	return schema
}

// parseArrayType 解析数组类型
func (tr *TypeResolver) parseArrayType(arrayType *ast.ArrayType) *OpenAPISchema {
	return &OpenAPISchema{
		Type:  "array",
		Items: tr.parseFieldType(arrayType.Elt),
	}
}

// parsePointerType 解析指针类型
func (tr *TypeResolver) parsePointerType(starExpr *ast.StarExpr) *OpenAPISchema {
	return tr.parseFieldType(starExpr.X)
}

// parseInterfaceType 解析接口类型
func (tr *TypeResolver) parseInterfaceType(interfaceName string, interfaceType *ast.InterfaceType) *OpenAPISchema {
	return &OpenAPISchema{
		Type:        "object",
		Description: tr.extractInterfaceComment(interfaceType),
	}
}

// parseFieldType 解析字段类型
func (tr *TypeResolver) parseFieldType(expr ast.Expr) *OpenAPISchema {
	switch t := expr.(type) {
	case *ast.Ident:
		return tr.parseBasicType(t.Name)
	case *ast.StarExpr:
		return tr.parseFieldType(t.X)
	case *ast.ArrayType:
		return &OpenAPISchema{
			Type:  "array",
			Items: tr.parseFieldType(t.Elt),
		}
	case *ast.SelectorExpr:
		// 包限定类型，如 models.User
		return &OpenAPISchema{
			Ref: "#/components/schemas/" + t.Sel.Name,
		}
	case *ast.MapType:
		return tr.parseMapType(t)
	default:
		return &OpenAPISchema{
			Type: "string",
		}
	}
}

// parseMapType 解析Map类型
func (tr *TypeResolver) parseMapType(mapType *ast.MapType) *OpenAPISchema {
	return &OpenAPISchema{
		Type: "object",
		AdditionalProperties: &OpenAPISchema{
			Type: "string", // 默认值，可以根据实际情况调整
		},
	}
}

// parseBasicType 解析基本类型
func (tr *TypeResolver) parseBasicType(typeName string) *OpenAPISchema {
	switch typeName {
	case "string":
		return &OpenAPISchema{Type: "string"}
	case "int", "int8", "int16", "int32", "int64":
		return &OpenAPISchema{Type: "integer"}
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return &OpenAPISchema{Type: "integer", Minimum: float64Ptr(0)}
	case "float32", "float64":
		return &OpenAPISchema{Type: "number"}
	case "bool":
		return &OpenAPISchema{Type: "boolean"}
	case "time.Time":
		return &OpenAPISchema{Type: "string", Format: "date-time"}
	case "[]byte":
		return &OpenAPISchema{Type: "string", Format: "byte"}
	case "error":
		return &OpenAPISchema{Type: "string"}
	default:
		// 可能是自定义类型，尝试解析
		return &OpenAPISchema{Type: "string"}
	}
}

// parseFieldTags 解析字段标签
func (tr *TypeResolver) parseFieldTags(schema *OpenAPISchema, tagValue string) *OpenAPISchema {
	// 解析json标签
	if jsonTag := tr.extractTagValue(tagValue, "json"); jsonTag != "" {
		if jsonTag == "-" {
			return nil // 忽略此字段
		}
		// 可以在这里处理json标签的其他选项
	}

	// 解析binding标签
	if bindingTag := tr.extractTagValue(tagValue, "binding"); bindingTag != "" {
		if strings.Contains(bindingTag, "required") {
			// 字段是必需的
		}
	}

	// 解析validate标签
	if validateTag := tr.extractTagValue(tagValue, "validate"); validateTag != "" {
		schema = tr.parseValidationTags(schema, validateTag)
	}

	// 解析example标签
	if exampleTag := tr.extractTagValue(tagValue, "example"); exampleTag != "" {
		schema.Example = exampleTag
	}

	// 解析default标签
	if defaultTag := tr.extractTagValue(tagValue, "default"); defaultTag != "" {
		schema.Default = defaultTag
	}

	return schema
}

// extractJSONFieldName 从JSON标签中提取字段名
func (tr *TypeResolver) extractJSONFieldName(tagValue string) string {
	jsonTag := tr.extractTagValue(tagValue, "json")
	if jsonTag == "" || jsonTag == "-" {
		return ""
	}

	// 处理逗号分隔的选项，如 "name,omitempty"
	parts := strings.Split(jsonTag, ",")
	return parts[0]
}

// parseValidationTags 解析验证标签
func (tr *TypeResolver) parseValidationTags(schema *OpenAPISchema, validateTag string) *OpenAPISchema {
	// 解析min/max长度
	if strings.Contains(validateTag, "min=") {
		if minValue := tr.extractNumericValue(validateTag, "min="); minValue != nil {
			if schema.Type == "string" {
				schema.MinLength = intPtr(int(*minValue))
			} else if schema.Type == "number" || schema.Type == "integer" {
				schema.Minimum = minValue
			}
		}
	}

	if strings.Contains(validateTag, "max=") {
		if maxValue := tr.extractNumericValue(validateTag, "max="); maxValue != nil {
			if schema.Type == "string" {
				schema.MaxLength = intPtr(int(*maxValue))
			} else if schema.Type == "number" || schema.Type == "integer" {
				schema.Maximum = maxValue
			}
		}
	}

	// 解析oneof
	if strings.HasPrefix(validateTag, "oneof=") {
		enumValues := tr.extractEnumValues(validateTag)
		if len(enumValues) > 0 {
			schema.Enum = enumValues
		}
	}

	// 解析email
	if strings.Contains(validateTag, "email") {
		schema.Format = "email"
	}

	// 解析url
	if strings.Contains(validateTag, "url") {
		schema.Format = "uri"
	}

	return schema
}

// extractTagValue 从标签字符串中提取指定标签的值
func (tr *TypeResolver) extractTagValue(tagValue, tagName string) string {
	// 移除引号
	tagValue = strings.Trim(tagValue, "`")

	// 查找标签
	start := strings.Index(tagValue, tagName+":")
	if start == -1 {
		return ""
	}

	start += len(tagName) + 1

	// 查找值的开始和结束
	start = strings.Index(tagValue[start:], "\"")
	if start == -1 {
		return ""
	}
	start++

	end := strings.Index(tagValue[start:], "\"")
	if end == -1 {
		return ""
	}

	return tagValue[start : start+end]
}

// extractNumericValue 提取数值
func (tr *TypeResolver) extractNumericValue(tag, prefix string) *float64 {
	start := strings.Index(tag, prefix)
	if start == -1 {
		return nil
	}

	start += len(prefix)
	end := start

	// 查找下一个分隔符
	for end < len(tag) && (tag[end] >= '0' && tag[end] <= '9' || tag[end] == '.') {
		end++
	}

	if start == end {
		return nil
	}

	value, err := strconv.ParseFloat(tag[start:end], 64)
	if err != nil {
		return nil
	}

	return &value
}

// extractEnumValues 提取枚举值
func (tr *TypeResolver) extractEnumValues(tag string) []interface{} {
	start := strings.Index(tag, "oneof=")
	if start == -1 {
		return nil
	}

	start += 7 // "oneof=" 的长度
	end := start

	// 查找下一个分隔符
	for end < len(tag) && tag[end] != ' ' && tag[end] != ',' {
		end++
	}

	valuesStr := tag[start:end]
	values := strings.Split(valuesStr, " ")

	result := make([]interface{}, 0, len(values))
	for _, value := range values {
		if value != "" {
			result = append(result, value)
		}
	}

	return result
}

// extractStructComment 提取结构体注释
func (tr *TypeResolver) extractStructComment(structType *ast.StructType) string {
	// 由于结构体注释通常存储在父节点中，我们需要通过其他方式获取
	// 通过分析结构体字段来推断注释
	if structType != nil && structType.Fields != nil {
		// 统计字段数量和类型
		fieldCount := len(structType.Fields.List)
		if fieldCount == 0 {
			return "空结构体"
		}

		// 分析字段类型，推断结构体用途
		var fieldTypes []string
		var hasID bool
		var hasTime bool
		var hasJSON bool

		for _, field := range structType.Fields.List {
			if field.Names != nil && len(field.Names) > 0 {
				fieldName := field.Names[0].Name
				fieldType := tr.extractFieldType(field.Type)
				fieldTypes = append(fieldTypes, fmt.Sprintf("%s %s", fieldName, fieldType))

				// 检查特殊字段
				if strings.Contains(strings.ToLower(fieldName), "id") {
					hasID = true
				}
				if strings.Contains(fieldType, "Time") {
					hasTime = true
				}
				if field.Tag != nil && strings.Contains(field.Tag.Value, "json:") {
					hasJSON = true
				}
			}
		}

		// 根据字段特征推断结构体用途
		if hasID && hasTime {
			return "数据模型结构体（包含ID和时间戳）"
		} else if hasID {
			return "数据模型结构体（包含ID）"
		} else if hasJSON {
			return "API请求/响应结构体"
		} else if fieldCount > 5 {
			return "复杂数据结构体"
		} else if fieldCount > 0 {
			return fmt.Sprintf("包含 %d 个字段的结构体", fieldCount)
		}
	}

	return "结构体"
}

// extractFieldType 提取字段类型
func (tr *TypeResolver) extractFieldType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + tr.extractFieldType(t.X)
	case *ast.ArrayType:
		return "[]" + tr.extractFieldType(t.Elt)
	case *ast.SelectorExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name + "." + t.Sel.Name
		}
		return tr.extractFieldType(t.X) + "." + t.Sel.Name
	default:
		return "interface{}"
	}
}

// extractInterfaceComment 提取接口注释
func (tr *TypeResolver) extractInterfaceComment(interfaceType *ast.InterfaceType) string {
	// 接口注释同样存储在父节点中
	return ""
}

// isFieldRequired 检查字段是否必需
func (tr *TypeResolver) isFieldRequired(field *ast.Field) bool {
	if field.Tag == nil {
		return false
	}

	tagValue := field.Tag.Value
	return strings.Contains(tagValue, "binding:\"required\"") ||
		strings.Contains(tagValue, "validate:\"required\"")
}

// parseTypeName 解析类型名称
func (tr *TypeResolver) parseTypeName(typeName string) (string, string) {
	// 解析类型名称，如 "models.User" -> ("models", "User")
	parts := strings.Split(typeName, ".")
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}
