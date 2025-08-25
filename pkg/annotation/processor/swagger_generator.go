package processor

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/grain-framework/grain/pkg/annotation/registry"
	"github.com/grain-framework/grain/pkg/annotation/types"
)

// SwaggerGenerator Swagger文档生成器
type SwaggerGenerator struct {
	registry     registry.Registry
	outputDir    string
	typeCache    map[string]*OpenAPISchema
	packages     map[string]*ast.Package
	typeResolver *TypeResolver
}

// OpenAPISchema OpenAPI Schema定义
type OpenAPISchema struct {
	Type                 string                    `json:"type,omitempty"`
	Format               string                    `json:"format,omitempty"`
	Description          string                    `json:"description,omitempty"`
	Example              interface{}               `json:"example,omitempty"`
	Default              interface{}               `json:"default,omitempty"`
	Required             []string                  `json:"required,omitempty"`
	Properties           map[string]*OpenAPISchema `json:"properties,omitempty"`
	Items                *OpenAPISchema            `json:"items,omitempty"`
	Ref                  string                    `json:"$ref,omitempty"`
	Enum                 []interface{}             `json:"enum,omitempty"`
	MinLength            *int                      `json:"minLength,omitempty"`
	MaxLength            *int                      `json:"maxLength,omitempty"`
	Minimum              *float64                  `json:"minimum,omitempty"`
	Maximum              *float64                  `json:"maximum,omitempty"`
	Pattern              string                    `json:"pattern,omitempty"`
	AdditionalProperties *OpenAPISchema            `json:"additionalProperties,omitempty"`
}

// OpenAPIPathItem OpenAPI路径项
type OpenAPIPathItem struct {
	Get     *OpenAPIOperation `json:"get,omitempty"`
	Post    *OpenAPIOperation `json:"post,omitempty"`
	Put     *OpenAPIOperation `json:"put,omitempty"`
	Delete  *OpenAPIOperation `json:"delete,omitempty"`
	Patch   *OpenAPIOperation `json:"patch,omitempty"`
	Options *OpenAPIOperation `json:"options,omitempty"`
	Head    *OpenAPIOperation `json:"head,omitempty"`
}

// OpenAPIOperation OpenAPI操作定义
type OpenAPIOperation struct {
	Tags        []string                    `json:"tags,omitempty"`
	Summary     string                      `json:"summary,omitempty"`
	Description string                      `json:"description,omitempty"`
	OperationID string                      `json:"operationId,omitempty"`
	Parameters  []*OpenAPIParameter         `json:"parameters,omitempty"`
	RequestBody *OpenAPIRequestBody         `json:"requestBody,omitempty"`
	Responses   map[string]*OpenAPIResponse `json:"responses"`
	Security    []map[string][]string       `json:"security,omitempty"`
}

// OpenAPIParameter OpenAPI参数定义
type OpenAPIParameter struct {
	Name        string         `json:"name"`
	In          string         `json:"in"`
	Description string         `json:"description,omitempty"`
	Required    bool           `json:"required,omitempty"`
	Schema      *OpenAPISchema `json:"schema"`
	Example     interface{}    `json:"example,omitempty"`
}

// OpenAPIRequestBody OpenAPI请求体定义
type OpenAPIRequestBody struct {
	Description string                     `json:"description,omitempty"`
	Required    bool                       `json:"required,omitempty"`
	Content     map[string]*OpenAPIContent `json:"content"`
}

// OpenAPIContent OpenAPI内容定义
type OpenAPIContent struct {
	Schema *OpenAPISchema `json:"schema"`
}

// OpenAPIResponse OpenAPI响应定义
type OpenAPIResponse struct {
	Description string                     `json:"description"`
	Content     map[string]*OpenAPIContent `json:"content,omitempty"`
}

// OpenAPISpec OpenAPI规范文档
type OpenAPISpec struct {
	OpenAPI    string                      `json:"openapi"`
	Info       *OpenAPIInfo                `json:"info"`
	Servers    []*OpenAPIServer            `json:"servers,omitempty"`
	Paths      map[string]*OpenAPIPathItem `json:"paths"`
	Components *OpenAPIComponents          `json:"components"`
}

// OpenAPIInfo OpenAPI信息
type OpenAPIInfo struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version"`
}

// OpenAPIServer OpenAPI服务器
type OpenAPIServer struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// OpenAPIComponents OpenAPI组件
type OpenAPIComponents struct {
	Schemas         map[string]*OpenAPISchema         `json:"schemas,omitempty"`
	SecuritySchemes map[string]*OpenAPISecurityScheme `json:"securitySchemes,omitempty"`
}

// OpenAPISecurityScheme OpenAPI安全方案
type OpenAPISecurityScheme struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// NewSwaggerGenerator 创建新的Swagger生成器
func NewSwaggerGenerator(registry registry.Registry, outputDir string) *SwaggerGenerator {
	return &SwaggerGenerator{
		registry:     registry,
		outputDir:    outputDir,
		typeCache:    make(map[string]*OpenAPISchema),
		packages:     make(map[string]*ast.Package),
		typeResolver: NewTypeResolver(outputDir),
	}
}

// GenerateSwaggerSpec 生成Swagger规范文档
func (sg *SwaggerGenerator) GenerateSwaggerSpec() (*OpenAPISpec, error) {
	// 1. 扫描所有注解
	annotations := sg.registry.GetAll()

	// 2. 解析控制器和路由信息
	paths, err := sg.parsePaths(annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to parse paths: %w", err)
	}

	// 3. 解析所有引用的类型
	schemas, err := sg.parseSchemas(annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to parse schemas: %w", err)
	}

	// 4. 构建OpenAPI规范
	spec := &OpenAPISpec{
		OpenAPI: "3.0.0",
		Info: &OpenAPIInfo{
			Title:       "gRain Framework API",
			Description: "基于gRain框架的API文档",
			Version:     "1.0.0",
		},
		Servers: []*OpenAPIServer{
			{
				URL:         "http://localhost:8080",
				Description: "开发环境",
			},
		},
		Paths: paths,
		Components: &OpenAPIComponents{
			Schemas:         schemas,
			SecuritySchemes: sg.buildSecuritySchemes(),
		},
	}

	return spec, nil
}

// parsePaths 解析路径信息
func (sg *SwaggerGenerator) parsePaths(annotations []types.Annotation) (map[string]*OpenAPIPathItem, error) {
	paths := make(map[string]*OpenAPIPathItem)

	// 按控制器分组
	controllers := make(map[string]*ControllerInfo)
	routes := make(map[string][]*RouteInfo)

	// 收集控制器信息
	for _, anno := range annotations {
		if anno.GetType() == types.ControllerType {
			pathPrefix := types.GetStringAttribute(anno.GetAttributes(), "path", "")
			controllers[anno.GetTargetName()] = &ControllerInfo{
				StructName: anno.GetTargetName(),
				PathPrefix: pathPrefix,
			}
		}
	}

	// 收集路由信息
	for _, anno := range annotations {
		if anno.GetType() == types.RouteType {
			routeInfo := &RouteInfo{
				Method:       types.GetStringAttribute(anno.GetAttributes(), "method", ""),
				Path:         types.GetStringAttribute(anno.GetAttributes(), "path", ""),
				FuncName:     anno.GetTargetName(),
				ReceiverType: sg.findControllerName(anno),
			}
			routes[routeInfo.ReceiverType] = append(routes[routeInfo.ReceiverType], routeInfo)
		}
	}

	// 构建路径
	for controllerName, controller := range controllers {
		controllerRoutes := routes[controllerName]
		for _, route := range controllerRoutes {
			fullPath := sg.buildFullPath(controller.PathPrefix, route.Path)

			if _, exists := paths[fullPath]; !exists {
				paths[fullPath] = &OpenAPIPathItem{}
			}

			operation := sg.buildOperation(route, controller)
			sg.setPathMethod(paths[fullPath], route.Method, operation)
		}
	}

	return paths, nil
}

// parseSchemas 解析所有引用的类型
func (sg *SwaggerGenerator) parseSchemas(annotations []types.Annotation) (map[string]*OpenAPISchema, error) {
	schemas := make(map[string]*OpenAPISchema)

	// 收集所有引用的类型
	referencedTypes := make(map[string]bool)

	for _, anno := range annotations {
		if anno.GetType() == types.RouteType {
			// 解析请求体类型
			if bindModel := types.GetStringAttribute(anno.GetAttributes(), "model", ""); bindModel != "" {
				referencedTypes[bindModel] = true
			}

			// 解析响应类型
			if responseType := sg.extractResponseType(anno); responseType != "" {
				referencedTypes[responseType] = true
			}
		}
	}

	// 解析每个引用的类型
	for typeName := range referencedTypes {
		schema, err := sg.typeResolver.ResolveType(typeName)
		if err != nil {
			fmt.Printf("Warning: failed to resolve type %s: %v\n", typeName, err)
			continue
		}

		if schema != nil {
			// 提取类型名称（去掉包前缀）
			shortName := sg.extractShortTypeName(typeName)
			schemas[shortName] = schema
		}
	}

	return schemas, nil
}

// resolveType 解析Go类型为OpenAPI Schema
func (sg *SwaggerGenerator) resolveType(typeName string) (*OpenAPISchema, error) {
	// 检查缓存
	if schema, exists := sg.typeCache[typeName]; exists {
		return schema, nil
	}

	// 解析类型名称
	packagePath, structName := sg.parseTypeName(typeName)
	if packagePath == "" || structName == "" {
		return nil, fmt.Errorf("invalid type name: %s", typeName)
	}

	// 查找包
	pkg, err := sg.findPackage(packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to find package %s: %w", packagePath, err)
	}

	// 查找结构体定义
	structType, err := sg.findStructType(pkg, structName)
	if err != nil {
		return nil, fmt.Errorf("failed to find struct %s: %w", structName, err)
	}

	// 解析结构体
	schema := sg.parseStructType(structName, structType)

	// 缓存结果
	sg.typeCache[typeName] = schema

	return schema, nil
}

// findPackage 查找包
func (sg *SwaggerGenerator) findPackage(packagePath string) (*ast.Package, error) {
	if pkg, exists := sg.packages[packagePath]; exists {
		return pkg, nil
	}

	// 解析包
	pkgs, err := parser.ParseDir(token.NewFileSet(), packagePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// 取第一个包
	for _, pkg := range pkgs {
		sg.packages[packagePath] = pkg
		return pkg, nil
	}

	return nil, fmt.Errorf("no package found in %s", packagePath)
}

// findStructType 查找结构体类型
func (sg *SwaggerGenerator) findStructType(pkg *ast.Package, structName string) (*ast.StructType, error) {
	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			if genDecl, ok := decl.(*ast.GenDecl); ok {
				for _, spec := range genDecl.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						if typeSpec.Name.Name == structName {
							if structType, ok := typeSpec.Type.(*ast.StructType); ok {
								return structType, nil
							}
						}
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("struct %s not found", structName)
}

// parseStructType 解析结构体类型
func (sg *SwaggerGenerator) parseStructType(structName string, structType *ast.StructType) *OpenAPISchema {
	schema := &OpenAPISchema{
		Type:        "object",
		Properties:  make(map[string]*OpenAPISchema),
		Required:    make([]string, 0),
		Description: sg.extractStructComment(structType),
	}

	// 解析字段
	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			continue
		}

		fieldName := field.Names[0].Name
		fieldSchema := sg.parseFieldType(field.Type)

		// 解析标签
		if field.Tag != nil {
			fieldSchema = sg.parseFieldTags(fieldSchema, field.Tag.Value)
		}

		// 解析注释
		if field.Doc != nil {
			fieldSchema.Description = strings.TrimSpace(field.Doc.Text())
		}

		// 检查是否必需
		if sg.isFieldRequired(field) {
			schema.Required = append(schema.Required, fieldName)
		}

		schema.Properties[fieldName] = fieldSchema
	}

	return schema
}

// parseFieldType 解析字段类型
func (sg *SwaggerGenerator) parseFieldType(expr ast.Expr) *OpenAPISchema {
	switch t := expr.(type) {
	case *ast.Ident:
		return sg.parseBasicType(t.Name)
	case *ast.StarExpr:
		return sg.parseFieldType(t.X)
	case *ast.ArrayType:
		return &OpenAPISchema{
			Type:  "array",
			Items: sg.parseFieldType(t.Elt),
		}
	case *ast.SelectorExpr:
		// 包限定类型，如 models.User
		return &OpenAPISchema{
			Ref: "#/components/schemas/" + t.Sel.Name,
		}
	default:
		return &OpenAPISchema{
			Type: "string",
		}
	}
}

// parseBasicType 解析基本类型
func (sg *SwaggerGenerator) parseBasicType(typeName string) *OpenAPISchema {
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
	default:
		return &OpenAPISchema{Type: "string"}
	}
}

// parseFieldTags 解析字段标签
func (sg *SwaggerGenerator) parseFieldTags(schema *OpenAPISchema, tagValue string) *OpenAPISchema {
	// 解析json标签
	if jsonTag := sg.extractTagValue(tagValue, "json"); jsonTag != "" {
		if jsonTag == "-" {
			return nil // 忽略此字段
		}
		// 可以在这里处理json标签的其他选项
	}

	// 解析binding标签
	if bindingTag := sg.extractTagValue(tagValue, "binding"); bindingTag != "" {
		if strings.Contains(bindingTag, "required") {
			// 字段是必需的
		}
	}

	// 解析validate标签
	if validateTag := sg.extractTagValue(tagValue, "validate"); validateTag != "" {
		schema = sg.parseValidationTags(schema, validateTag)
	}

	// 解析example标签
	if exampleTag := sg.extractTagValue(tagValue, "example"); exampleTag != "" {
		schema.Example = exampleTag
	}

	return schema
}

// parseValidationTags 解析验证标签
func (sg *SwaggerGenerator) parseValidationTags(schema *OpenAPISchema, validateTag string) *OpenAPISchema {
	// 解析min/max长度
	if strings.Contains(validateTag, "min=") {
		if minValue := sg.extractNumericValue(validateTag, "min="); minValue != nil {
			if schema.Type == "string" {
				schema.MinLength = intPtr(int(*minValue))
			} else if schema.Type == "number" || schema.Type == "integer" {
				schema.Minimum = minValue
			}
		}
	}

	if strings.Contains(validateTag, "max=") {
		if maxValue := sg.extractNumericValue(validateTag, "max="); maxValue != nil {
			if schema.Type == "string" {
				schema.MaxLength = intPtr(int(*maxValue))
			} else if schema.Type == "number" || schema.Type == "integer" {
				schema.Maximum = maxValue
			}
		}
	}

	// 解析oneof
	if strings.HasPrefix(validateTag, "oneof=") {
		enumValues := sg.extractEnumValues(validateTag)
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

// 辅助方法
func (sg *SwaggerGenerator) findControllerName(anno types.Annotation) string {
	// 从注解中查找控制器名称
	// 根据注解的关联关系来确定控制器名称
	if anno.GetTargetType() == types.TypeTarget {
		return anno.GetTargetName()
	}

	// 如果是方法注解，尝试查找对应的控制器
	// 通过注解的目标信息推断控制器名称
	if anno.GetTargetType() == types.MethodTarget {
		// 从方法名推断控制器名
		methodName := anno.GetTargetName()
		if strings.HasSuffix(methodName, "Handler") {
			// 如：UserHandler -> User
			return strings.TrimSuffix(methodName, "Handler")
		} else if strings.HasSuffix(methodName, "Action") {
			// 如：UserAction -> User
			return strings.TrimSuffix(methodName, "Action")
		} else if strings.HasSuffix(methodName, "Func") {
			// 如：UserFunc -> User
			return strings.TrimSuffix(methodName, "Func")
		}

		// 尝试从注解位置推断
		if commentAnn, ok := anno.(*types.CommentAnnotation); ok {
			// 从文件路径推断控制器名
			if commentAnn.Position.Filename != "" {
				fileName := filepath.Base(commentAnn.Position.Filename)
				if strings.Contains(fileName, "_controller.go") {
					// 如：user_controller.go -> User
					parts := strings.Split(fileName, "_")
					if len(parts) > 0 {
						return strings.Title(parts[0])
					}
				} else if strings.Contains(fileName, "controller.go") {
					// 如：usercontroller.go -> User
					parts := strings.Split(fileName, "controller.go")
					if len(parts) > 0 {
						return strings.Title(parts[0])
					}
				}
			}
		}

		// 如果无法推断，使用默认值
		return "DefaultController"
	}

	return "DefaultController"
}

func (sg *SwaggerGenerator) buildFullPath(prefix, path string) string {
	if prefix == "" {
		return path
	}
	if path == "" {
		return prefix
	}

	// 确保路径之间只有一个斜杠
	if strings.HasSuffix(prefix, "/") && strings.HasPrefix(path, "/") {
		return prefix + path[1:]
	}
	if !strings.HasSuffix(prefix, "/") && !strings.HasPrefix(path, "/") {
		return prefix + "/" + path
	}
	return prefix + path
}

func (sg *SwaggerGenerator) buildOperation(route *RouteInfo, controller *ControllerInfo) *OpenAPIOperation {
	operation := &OpenAPIOperation{
		Tags:        []string{controller.StructName},
		Summary:     route.FuncName, // 使用函数名作为摘要，因为RouteInfo中没有Summary字段
		OperationID: route.FuncName,
		Responses:   sg.buildDefaultResponses(),
	}

	// 添加请求体信息
	if requestBody := sg.buildRequestBody(route); requestBody != nil {
		operation.RequestBody = requestBody
	}

	// 添加参数信息
	if params := sg.buildParameters(route); len(params) > 0 {
		operation.Parameters = params
	}

	return operation
}

func (sg *SwaggerGenerator) setPathMethod(pathItem *OpenAPIPathItem, method string, operation *OpenAPIOperation) {
	switch strings.ToUpper(method) {
	case "GET":
		pathItem.Get = operation
	case "POST":
		pathItem.Post = operation
	case "PUT":
		pathItem.Put = operation
	case "DELETE":
		pathItem.Delete = operation
	case "PATCH":
		pathItem.Patch = operation
	}
}

func (sg *SwaggerGenerator) buildDefaultResponses() map[string]*OpenAPIResponse {
	return map[string]*OpenAPIResponse{
		"200": {
			Description: "成功",
		},
		"400": {
			Description: "请求参数错误",
		},
		"500": {
			Description: "服务器内部错误",
		},
	}
}

func (sg *SwaggerGenerator) buildSecuritySchemes() map[string]*OpenAPISecurityScheme {
	return map[string]*OpenAPISecurityScheme{
		"bearerAuth": {
			Type:        "http",
			Description: "Bearer token认证",
		},
	}
}

func (sg *SwaggerGenerator) extractResponseType(anno types.Annotation) string {
	// 从注解中提取响应类型
	if commentAnn, ok := anno.(*types.CommentAnnotation); ok {
		// 从注解内容中提取响应类型
		content := commentAnn.Raw

		// 查找响应类型标记
		if strings.Contains(content, "response:") {
			parts := strings.Split(content, "response:")
			if len(parts) > 1 {
				responseType := strings.TrimSpace(strings.Split(parts[1], " ")[0])
				return responseType
			}
		}

		// 查找返回类型标记
		if strings.Contains(content, "returns:") {
			parts := strings.Split(content, "returns:")
			if len(parts) > 1 {
				returnType := strings.TrimSpace(strings.Split(parts[1], " ")[0])
				return returnType
			}
		}

		// 查找模型标记
		if strings.Contains(content, "model:") {
			parts := strings.Split(content, "model:")
			if len(parts) > 1 {
				modelType := strings.TrimSpace(strings.Split(parts[1], " ")[0])
				return modelType
			}
		}
	}

	// 如果无法提取，返回默认值
	return "interface{}"
}

func (sg *SwaggerGenerator) parseTypeName(typeName string) (string, string) {
	// 解析类型名称，如 "models.User" -> ("models", "User")
	parts := strings.Split(typeName, ".")
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func (sg *SwaggerGenerator) extractShortTypeName(typeName string) string {
	// 提取短类型名称，去掉包前缀
	parts := strings.Split(typeName, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return typeName
}

// extractStructComment 提取结构体注释
func (sg *SwaggerGenerator) extractStructComment(structType *ast.StructType) string {
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
		for _, field := range structType.Fields.List {
			if field.Names != nil && len(field.Names) > 0 {
				fieldName := field.Names[0].Name
				fieldType := sg.extractFieldType(field.Type)
				fieldTypes = append(fieldTypes, fmt.Sprintf("%s %s", fieldName, fieldType))
			}
		}

		// 根据字段特征推断结构体用途
		if len(fieldTypes) > 0 {
			// 检查是否是数据模型
			if sg.isDataModel(fieldTypes) {
				return "数据模型结构体"
			}
			// 检查是否是请求结构体
			if sg.isRequestStruct(fieldTypes) {
				return "请求参数结构体"
			}
			// 检查是否是响应结构体
			if sg.isResponseStruct(fieldTypes) {
				return "响应数据结构体"
			}
			// 检查是否是配置结构体
			if sg.isConfigStruct(fieldTypes) {
				return "配置结构体"
			}
		}

		return fmt.Sprintf("包含 %d 个字段的结构体", fieldCount)
	}

	return "结构体"
}

// extractFieldType 提取字段类型
func (sg *SwaggerGenerator) extractFieldType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + sg.extractFieldType(t.X)
	case *ast.ArrayType:
		return "[]" + sg.extractFieldType(t.Elt)
	case *ast.SelectorExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name + "." + t.Sel.Name
		}
		return sg.extractFieldType(t.X) + "." + t.Sel.Name
	default:
		return "interface{}"
	}
}

// isDataModel 检查是否是数据模型
func (sg *SwaggerGenerator) isDataModel(fieldTypes []string) bool {
	// 检查是否包含常见的数据库字段
	dataModelKeywords := []string{"ID", "Id", "id", "CreatedAt", "UpdatedAt", "DeletedAt"}
	for _, fieldType := range fieldTypes {
		for _, keyword := range dataModelKeywords {
			if strings.Contains(fieldType, keyword) {
				return true
			}
		}
	}
	return false
}

// isRequestStruct 检查是否是请求结构体
func (sg *SwaggerGenerator) isRequestStruct(fieldTypes []string) bool {
	// 检查是否包含常见的请求字段
	requestKeywords := []string{"binding", "validate", "json", "xml"}
	for _, fieldType := range fieldTypes {
		for _, keyword := range requestKeywords {
			if strings.Contains(fieldType, keyword) {
				return true
			}
		}
	}
	return false
}

// isResponseStruct 检查是否是响应结构体
func (sg *SwaggerGenerator) isResponseStruct(fieldTypes []string) bool {
	// 检查是否包含常见的响应字段
	responseKeywords := []string{"Code", "Message", "Data", "Status"}
	for _, fieldType := range fieldTypes {
		for _, keyword := range responseKeywords {
			if strings.Contains(fieldType, keyword) {
				return true
			}
		}
	}
	return false
}

// isConfigStruct 检查是否是配置结构体
func (sg *SwaggerGenerator) isConfigStruct(fieldTypes []string) bool {
	// 检查是否包含常见的配置字段
	configKeywords := []string{"Config", "Settings", "Options", "Env"}
	for _, fieldType := range fieldTypes {
		for _, keyword := range configKeywords {
			if strings.Contains(fieldType, keyword) {
				return true
			}
		}
	}
	return false
}

func (sg *SwaggerGenerator) isFieldRequired(field *ast.Field) bool {
	// 检查字段是否必需
	if field.Tag == nil {
		return false
	}

	tagValue := field.Tag.Value
	return strings.Contains(tagValue, "binding:\"required\"") ||
		strings.Contains(tagValue, "validate:\"required\"")
}

func (sg *SwaggerGenerator) extractTagValue(tagValue, tagName string) string {
	// 从标签字符串中提取指定标签的值
	// 移除反引号
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
func (sg *SwaggerGenerator) extractNumericValue(tag, prefix string) *float64 {
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
func (sg *SwaggerGenerator) extractEnumValues(tag string) []interface{} {
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

// 移除重复的结构体定义，因为它们已经在route_processor.go中定义了

// buildRequestBody 构建请求体
func (sg *SwaggerGenerator) buildRequestBody(route *RouteInfo) *OpenAPIRequestBody {
	// 从注解中提取请求体信息
	if route.BindModel == "" {
		return nil
	}

	// 解析绑定模型类型
	schema, err := sg.typeResolver.ResolveType(route.BindModel)
	if err != nil {
		// 如果解析失败，返回默认的object类型
		schema = &OpenAPISchema{
			Type: "object",
		}
	}

	return &OpenAPIRequestBody{
		Description: fmt.Sprintf("请求体 - %s", route.BindModel),
		Required:    true,
		Content: map[string]*OpenAPIContent{
			"application/json": {
				Schema: schema,
			},
		},
	}
}

// buildParameters 构建参数
func (sg *SwaggerGenerator) buildParameters(route *RouteInfo) []*OpenAPIParameter {
	var parameters []*OpenAPIParameter

	// 解析路径参数
	if strings.Contains(route.Path, ":") {
		pathParams := sg.extractPathParameters(route.Path)
		for _, param := range pathParams {
			parameters = append(parameters, &OpenAPIParameter{
				Name:        param,
				In:          "path",
				Required:    true,
				Description: fmt.Sprintf("路径参数: %s", param),
				Schema: &OpenAPISchema{
					Type: "string",
				},
			})
		}
	}

	// 解析查询参数（对于GET请求）
	if route.Method == "GET" {
		// 智能添加查询参数
		parameters = append(parameters, sg.buildCommonQueryParameters(route)...)

		// 根据路由路径和绑定模型添加特定参数
		if route.BindModel != "" {
			parameters = append(parameters, sg.buildModelSpecificParameters(route)...)
		}
	}

	return parameters
}

// extractPathParameters 提取路径参数
func (sg *SwaggerGenerator) extractPathParameters(path string) []string {
	var params []string
	parts := strings.Split(path, "/")

	for _, part := range parts {
		if strings.HasPrefix(part, ":") {
			paramName := strings.TrimPrefix(part, ":")
			params = append(params, paramName)
		}
	}

	return params
}

func float64Ptr(v float64) *float64 {
	return &v
}

func intPtr(v int) *int {
	return &v
}

// buildCommonQueryParameters 构建通用查询参数
func (sg *SwaggerGenerator) buildCommonQueryParameters(route *RouteInfo) []*OpenAPIParameter {
	var parameters []*OpenAPIParameter

	// 添加分页参数
	parameters = append(parameters, &OpenAPIParameter{
		Name:        "page",
		In:          "query",
		Required:    false,
		Description: "页码",
		Schema: &OpenAPISchema{
			Type:    "integer",
			Default: 1,
		},
	})

	parameters = append(parameters, &OpenAPIParameter{
		Name:        "size",
		In:          "query",
		Required:    false,
		Description: "每页大小",
		Schema: &OpenAPISchema{
			Type:    "integer",
			Default: 10,
		},
	})

	// 添加排序参数
	parameters = append(parameters, &OpenAPIParameter{
		Name:        "sort",
		In:          "query",
		Required:    false,
		Description: "排序字段",
		Schema: &OpenAPISchema{
			Type: "string",
		},
	})

	parameters = append(parameters, &OpenAPIParameter{
		Name:        "order",
		In:          "query",
		Required:    false,
		Description: "排序方向 (asc/desc)",
		Schema: &OpenAPISchema{
			Type:    "string",
			Default: "asc",
		},
	})

	// 添加搜索参数
	parameters = append(parameters, &OpenAPIParameter{
		Name:        "q",
		In:          "query",
		Required:    false,
		Description: "搜索关键词",
		Schema: &OpenAPISchema{
			Type: "string",
		},
	})

	return parameters
}

// buildModelSpecificParameters 构建模型特定的查询参数
func (sg *SwaggerGenerator) buildModelSpecificParameters(route *RouteInfo) []*OpenAPIParameter {
	var parameters []*OpenAPIParameter

	// 根据绑定模型类型添加特定参数
	if strings.Contains(route.BindModel, "User") {
		// 用户相关参数
		parameters = append(parameters, &OpenAPIParameter{
			Name:        "role",
			In:          "query",
			Required:    false,
			Description: "用户角色",
			Schema: &OpenAPISchema{
				Type: "string",
			},
		})

		parameters = append(parameters, &OpenAPIParameter{
			Name:        "status",
			In:          "query",
			Required:    false,
			Description: "用户状态",
			Schema: &OpenAPISchema{
				Type: "string",
			},
		})
	}

	if strings.Contains(route.BindModel, "Product") {
		// 产品相关参数
		parameters = append(parameters, &OpenAPIParameter{
			Name:        "category",
			In:          "query",
			Required:    false,
			Description: "产品分类",
			Schema: &OpenAPISchema{
				Type: "string",
			},
		})

		parameters = append(parameters, &OpenAPIParameter{
			Name:        "price_min",
			In:          "query",
			Required:    false,
			Description: "最低价格",
			Schema: &OpenAPISchema{
				Type: "number",
			},
		})

		parameters = append(parameters, &OpenAPIParameter{
			Name:        "price_max",
			In:          "query",
			Required:    false,
			Description: "最高价格",
			Schema: &OpenAPISchema{
				Type: "number",
			},
		})
	}

	if strings.Contains(route.BindModel, "Order") {
		// 订单相关参数
		parameters = append(parameters, &OpenAPIParameter{
			Name:        "status",
			In:          "query",
			Required:    false,
			Description: "订单状态",
			Schema: &OpenAPISchema{
				Type: "string",
			},
		})

		parameters = append(parameters, &OpenAPIParameter{
			Name:        "date_from",
			In:          "query",
			Required:    false,
			Description: "开始日期",
			Schema: &OpenAPISchema{
				Type:   "string",
				Format: "date",
			},
		})

		parameters = append(parameters, &OpenAPIParameter{
			Name:        "date_to",
			In:          "query",
			Required:    false,
			Description: "结束日期",
			Schema: &OpenAPISchema{
				Type:   "string",
				Format: "date",
			},
		})
	}

	return parameters
}
