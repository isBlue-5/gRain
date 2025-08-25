// Package processor 提供注解处理器的实现
package processor

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"

	"github.com/isBlue-5/grain/pkg/annotation/registry"
	anntypes "github.com/isBlue-5/grain/pkg/annotation/types"
)

// CRUDProcessor CRUD代码生成处理器
type CRUDProcessor struct {
	// 注解注册中心
	registry registry.Registry

	// 注解前缀
	prefix string

	// 输出目录
	outputDir string

	// 是否启用详细日志
	verbose bool
}

// NewCRUDProcessor 创建CRUD代码生成处理器
func NewCRUDProcessor(reg registry.Registry, prefix, outputDir string) *CRUDProcessor {
	return &CRUDProcessor{
		registry:  reg,
		prefix:    prefix,
		outputDir: outputDir,
		verbose:   false,
	}
}

// WithVerbose 设置是否启用详细日志
func (p *CRUDProcessor) WithVerbose(verbose bool) *CRUDProcessor {
	p.verbose = verbose
	return p
}

// EntityInfo 实体信息
type EntityInfo struct {
	// 实体名称
	Name string

	// 表名
	TableName string

	// 包路径
	PkgPath string

	// 字段信息
	Fields []FieldInfo

	// 主键字段
	PrimaryKey string

	// 原始注解
	Annotation anntypes.Annotation
}

// FieldInfo 字段信息
type FieldInfo struct {
	// 字段名称
	Name string

	// 字段类型
	Type string

	// 列名
	ColumnName string

	// 是否主键
	IsPrimaryKey bool

	// 是否必填
	IsRequired bool

	// 是否唯一
	IsUnique bool

	// 验证规则
	ValidationRules string

	// JSON标签
	JSONTag string
}

// ProcessCRUD 处理CRUD代码生成
func (p *CRUDProcessor) ProcessCRUD(pkgPaths []string) error {
	// 收集实体信息
	entities := p.collectEntityInfo()

	if len(entities) == 0 {
		if p.verbose {
			fmt.Println("未找到实体注解，跳过CRUD代码生成")
		}
		return nil
	}

	// 生成代码
	for _, entity := range entities {
		// 生成仓储代码
		if err := p.generateRepositoryCode(entity); err != nil {
			return fmt.Errorf("生成仓储代码失败: %w", err)
		}

		// 生成服务代码
		if err := p.generateServiceCode(entity); err != nil {
			return fmt.Errorf("生成服务代码失败: %w", err)
		}

		// 生成控制器代码
		if err := p.generateControllerCode(entity); err != nil {
			return fmt.Errorf("生成控制器代码失败: %w", err)
		}
	}

	return nil
}

// collectEntityInfo 收集实体信息
func (p *CRUDProcessor) collectEntityInfo() []EntityInfo {
	var entities []EntityInfo

	// 查找所有entity注解
	annotations := p.registry.FindByType(anntypes.EntityType)

	for _, anno := range annotations {
		// 确保目标是结构体
		if anno.GetTargetType() != anntypes.TypeTarget {
			continue
		}

		// 创建实体信息
		entity := EntityInfo{
			Name:       anno.GetTargetName(),
			TableName:  anntypes.GetStringAttribute(anno.GetAttributes(), "table", strings.ToLower(anno.GetTargetName())),
			PkgPath:    filepath.Dir(anno.GetPosition().Filename),
			Annotation: anno,
			Fields:     []FieldInfo{},
		}

		// 尝试解析字段
		if structAnno, ok := anno.(*anntypes.CommentAnnotation); ok && structAnno.Node != nil {
			if typeSpec, ok := structAnno.Node.(*ast.TypeSpec); ok {
				if structType, ok := typeSpec.Type.(*ast.StructType); ok {
					// 解析结构体字段
					entity.Fields = p.parseStructFields(structType)
				}
			}
		}

		// 查找主键
		for _, field := range entity.Fields {
			if field.IsPrimaryKey {
				entity.PrimaryKey = field.Name
				break
			}
		}

		// 如果没有明确的主键，默认使用ID字段
		if entity.PrimaryKey == "" {
			entity.PrimaryKey = "ID"
		}

		entities = append(entities, entity)
	}

	return entities
}

// parseStructFields 解析结构体字段
func (p *CRUDProcessor) parseStructFields(structType *ast.StructType) []FieldInfo {
	var fields []FieldInfo

	if structType.Fields == nil || len(structType.Fields.List) == 0 {
		return fields
	}

	for _, field := range structType.Fields.List {
		// 跳过匿名字段
		if field.Names == nil || len(field.Names) == 0 {
			continue
		}

		// 创建字段信息
		fieldInfo := FieldInfo{
			Name:       field.Names[0].Name,
			Type:       getTypeString(field.Type),
			ColumnName: strings.ToLower(field.Names[0].Name),
		}

		// 解析字段标签
		if field.Tag != nil {
			tag := strings.Trim(field.Tag.Value, "`")

			// 解析db标签
			if dbTag := extractTagValue(tag, "db"); dbTag != "" {
				parts := strings.Split(dbTag, ",")
				if len(parts) > 0 {
					fieldInfo.ColumnName = parts[0]
				}

				// 检查主键
				if contains(parts, "primaryKey") {
					fieldInfo.IsPrimaryKey = true
				}

				// 检查必填
				if contains(parts, "required") {
					fieldInfo.IsRequired = true
				}

				// 检查唯一
				if contains(parts, "unique") {
					fieldInfo.IsUnique = true
				}
			}

			// 解析validate标签
			if validateTag := extractTagValue(tag, "validate"); validateTag != "" {
				fieldInfo.ValidationRules = validateTag
			}

			// 解析json标签
			if jsonTag := extractTagValue(tag, "json"); jsonTag != "" {
				fieldInfo.JSONTag = jsonTag
			}
		}

		fields = append(fields, fieldInfo)
	}

	return fields
}

// 辅助函数：获取类型字符串
func getTypeString(expr ast.Expr) string {
	// 简化实现
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + getTypeString(t.X)
	case *ast.ArrayType:
		return "[]" + getTypeString(t.Elt)
	case *ast.SelectorExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name + "." + t.Sel.Name
		}
	}

	return "interface{}"
}

// 辅助函数：提取标签值
func extractTagValue(tag, key string) string {
	// 查找键
	keyStart := strings.Index(tag, key+":")
	if keyStart < 0 {
		return ""
	}

	// 找到值的开始位置（引号后）
	valueStart := keyStart + len(key) + 1
	if valueStart >= len(tag) {
		return ""
	}

	// 找到第一个引号
	if tag[valueStart] == '"' {
		valueStart++
	}

	// 找到值的结束位置（下一个引号）
	valueEnd := valueStart
	for valueEnd < len(tag) && tag[valueEnd] != '"' {
		valueEnd++
	}

	return tag[valueStart:valueEnd]
}

// 辅助函数：检查切片是否包含字符串
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// extractCRUDTagValue 从结构体标签中提取指定键的值
func extractCRUDTagValue(tag, key string) string {
	tagContent := reflect.StructTag(tag)
	if val, ok := tagContent.Lookup(key); ok {
		return strings.Split(val, ",")[0]
	}
	return ""
}

// generateRepositoryCode 生成仓储代码
func (p *CRUDProcessor) generateRepositoryCode(entity EntityInfo) error {
	// 准备模板数据
	data := map[string]interface{}{
		"PackageName":    "repository",
		"EntityName":     entity.Name,
		"EntityPkgPath":  filepath.Base(entity.PkgPath),
		"TableName":      entity.TableName,
		"Fields":         entity.Fields,
		"PrimaryKey":     entity.PrimaryKey,
		"PrimaryKeyType": getPrimaryKeyType(entity),
		"Imports":        collectImports(entity),
	}

	// 解析模板
	tmpl, err := template.New("repository").Parse(repositoryTemplate)
	if err != nil {
		return fmt.Errorf("解析仓储模板失败: %w", err)
	}

	// 生成代码
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("执行仓储模板失败: %w", err)
	}

	// 格式化代码
	formattedCode, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("格式化仓储代码失败: %w", err)
	}

	// 创建输出目录
	outDir := filepath.Join(p.outputDir, "repository")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("创建仓储输出目录失败: %w", err)
	}

	// 写入文件
	outFile := filepath.Join(outDir, fmt.Sprintf("%s_repository_gen.go", strings.ToLower(entity.Name)))
	if err := os.WriteFile(outFile, formattedCode, 0644); err != nil {
		return fmt.Errorf("写入仓储文件失败: %w", err)
	}

	if p.verbose {
		fmt.Printf("生成仓储代码: %s\n", outFile)
	}

	return nil
}

// generateServiceCode 生成服务代码
func (p *CRUDProcessor) generateServiceCode(entity EntityInfo) error {
	// 准备模板数据
	data := map[string]interface{}{
		"PackageName":    "service",
		"EntityName":     entity.Name,
		"EntityPkgPath":  filepath.Base(entity.PkgPath),
		"Fields":         entity.Fields,
		"PrimaryKey":     entity.PrimaryKey,
		"PrimaryKeyType": getPrimaryKeyType(entity),
		"Imports":        collectImports(entity),
	}

	// 解析模板
	tmpl, err := template.New("service").Parse(serviceTemplate)
	if err != nil {
		return fmt.Errorf("解析服务模板失败: %w", err)
	}

	// 生成代码
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("执行服务模板失败: %w", err)
	}

	// 格式化代码
	formattedCode, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("格式化服务代码失败: %w", err)
	}

	// 创建输出目录
	outDir := filepath.Join(p.outputDir, "service")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("创建服务输出目录失败: %w", err)
	}

	// 写入文件
	outFile := filepath.Join(outDir, fmt.Sprintf("%s_service_gen.go", strings.ToLower(entity.Name)))
	if err := os.WriteFile(outFile, formattedCode, 0644); err != nil {
		return fmt.Errorf("写入服务文件失败: %w", err)
	}

	if p.verbose {
		fmt.Printf("生成服务代码: %s\n", outFile)
	}

	return nil
}

// generateControllerCode 生成控制器代码
func (p *CRUDProcessor) generateControllerCode(entity EntityInfo) error {
	// 准备模板数据
	data := map[string]interface{}{
		"PackageName":    "controller",
		"EntityName":     entity.Name,
		"EntityPkgPath":  filepath.Base(entity.PkgPath),
		"Fields":         entity.Fields,
		"PrimaryKey":     entity.PrimaryKey,
		"PrimaryKeyType": getPrimaryKeyType(entity),
		"Imports":        collectImports(entity),
	}

	// 解析模板
	tmpl, err := template.New("controller").Parse(controllerTemplate)
	if err != nil {
		return fmt.Errorf("解析控制器模板失败: %w", err)
	}

	// 生成代码
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("执行控制器模板失败: %w", err)
	}

	// 格式化代码
	formattedCode, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("格式化控制器代码失败: %w", err)
	}

	// 创建输出目录
	outDir := filepath.Join(p.outputDir, "controller")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("创建控制器输出目录失败: %w", err)
	}

	// 写入文件
	outFile := filepath.Join(outDir, fmt.Sprintf("%s_controller.go", strings.ToLower(entity.Name)))
	if err := os.WriteFile(outFile, formattedCode, 0644); err != nil {
		return fmt.Errorf("写入控制器文件失败: %w", err)
	}

	if p.verbose {
		fmt.Printf("生成控制器代码: %s\n", outFile)
	}

	return nil
}

// 辅助函数：获取主键类型
func getPrimaryKeyType(entity EntityInfo) string {
	// 查找主键字段
	for _, field := range entity.Fields {
		if field.IsPrimaryKey || field.Name == entity.PrimaryKey {
			return field.Type
		}
	}

	// 默认主键类型
	return "int64"
}

// 辅助函数：收集导入
func collectImports(entity EntityInfo) []string {
	imports := []string{
		"github.com/gin-gonic/gin",
		"context",
		"errors",
	}

	// 添加实体包导入
	if filepath.Base(entity.PkgPath) != "model" {
		imports = append(imports, fmt.Sprintf("your_project/%s", entity.PkgPath))
	}

	// 检查字段类型是否需要额外导入
	for _, field := range entity.Fields {
		if strings.Contains(field.Type, "time.") {
			imports = append(imports, "time")
			break
		}
	}

	return imports
}

// 仓储模板
const repositoryTemplate = `// Code generated by gRain. DO NOT EDIT.
package repository

import (
	"context"
	"database/sql"
	"errors"
	{{range .Imports}}
	"{{.}}"
	{{end}}
	"your_project/{{.EntityPkgPath}}"
)

var (
	ErrNotFound = errors.New("record not found")
	ErrInvalidInput = errors.New("invalid input")
)

// {{.EntityName}}Repository 定义{{.EntityName}}仓储接口
// frame:repository
type {{.EntityName}}Repository interface {
	FindByID(ctx context.Context, id {{.PrimaryKeyType}}) (*{{.EntityPkgPath}}.{{.EntityName}}, error)
	FindAll(ctx context.Context) ([]*{{.EntityPkgPath}}.{{.EntityName}}, error)
	Create(ctx context.Context, entity *{{.EntityPkgPath}}.{{.EntityName}}) error
	Update(ctx context.Context, entity *{{.EntityPkgPath}}.{{.EntityName}}) error
	Delete(ctx context.Context, id {{.PrimaryKeyType}}) error
}

// {{.EntityName}}RepositoryImpl 实现{{.EntityName}}仓储接口
type {{.EntityName}}RepositoryImpl struct {
	db *sql.DB
}

// New{{.EntityName}}Repository 创建{{.EntityName}}仓储实现
// frame:inject
func New{{.EntityName}}Repository(db *sql.DB) {{.EntityName}}Repository {
	return &{{.EntityName}}RepositoryImpl{
		db: db,
	}
}

// FindByID 根据ID查找实体
func (r *{{.EntityName}}RepositoryImpl) FindByID(ctx context.Context, id {{.PrimaryKeyType}}) (*{{.EntityPkgPath}}.{{.EntityName}}, error) {
	query := ` + "`SELECT " +
	"{{- range $i, $field := .Fields}}{{if $i}}, {{end}}{{$field.ColumnName}}{{end -}} " +
	"FROM {{.TableName}} WHERE {{range $field := .Fields}}{{if $field.IsPrimaryKey}}{{$field.ColumnName}}{{end}}{{end}} = ?`" + `
	
	row := r.db.QueryRowContext(ctx, query, id)
	
	entity := &{{.EntityPkgPath}}.{{.EntityName}}{}
	err := row.Scan(
		{{- range $field := .Fields}}
		&entity.{{$field.Name}},
		{{- end}}
	)
	
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	
	if err != nil {
		return nil, err
	}
	
	return entity, nil
}

// FindAll 查询所有实体
func (r *{{.EntityName}}RepositoryImpl) FindAll(ctx context.Context) ([]*{{.EntityPkgPath}}.{{.EntityName}}, error) {
	query := ` + "`SELECT " +
	"{{- range $i, $field := .Fields}}{{if $i}}, {{end}}{{$field.ColumnName}}{{end -}} " +
	"FROM {{.TableName}}`" + `
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var result []*{{.EntityPkgPath}}.{{.EntityName}}
	
	for rows.Next() {
		entity := &{{.EntityPkgPath}}.{{.EntityName}}{}
		err := rows.Scan(
			{{- range $field := .Fields}}
			&entity.{{$field.Name}},
			{{- end}}
		)
		
		if err != nil {
			return nil, err
		}
		
		result = append(result, entity)
	}
	
	if err = rows.Err(); err != nil {
		return nil, err
	}
	
	return result, nil
}

// Create 创建新实体
func (r *{{.EntityName}}RepositoryImpl) Create(ctx context.Context, entity *{{.EntityPkgPath}}.{{.EntityName}}) error {
	if entity == nil {
		return ErrInvalidInput
	}
	
	query := ` + "`INSERT INTO {{.TableName}} (" +
	"{{- range $i, $field := .Fields}}{{if not $field.IsPrimaryKey}}{{if $i}}, {{end}}{{$field.ColumnName}}{{end}}{{end -}}" +
	") VALUES (" +
	"{{- range $i, $field := .Fields}}{{if not $field.IsPrimaryKey}}{{if $i}}, {{end}}?{{end}}{{end -}}" +
	")`" + `
	
	result, err := r.db.ExecContext(
		ctx,
		query,
		{{- range $field := .Fields}}
		{{- if not $field.IsPrimaryKey}}
		entity.{{$field.Name}},
		{{- end}}
		{{- end}}
	)
	
	if err != nil {
		return err
	}
	
	{{if eq .PrimaryKeyType "int64"}}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	
	entity.{{.PrimaryKey}} = id
	{{end}}
	
	return nil
}

// Update 更新实体
func (r *{{.EntityName}}RepositoryImpl) Update(ctx context.Context, entity *{{.EntityPkgPath}}.{{.EntityName}}) error {
	if entity == nil {
		return ErrInvalidInput
	}
	
	query := ` + "`UPDATE {{.TableName}} SET " +
	"{{- range $i, $field := .Fields}}{{if not $field.IsPrimaryKey}}{{if $i}}, {{end}}{{$field.ColumnName}} = ?{{end}}{{end -}} " +
	"WHERE {{range $field := .Fields}}{{if $field.IsPrimaryKey}}{{$field.ColumnName}}{{end}}{{end}} = ?`" + `
	
	_, err := r.db.ExecContext(
		ctx,
		query,
		{{- range $field := .Fields}}
		{{- if not $field.IsPrimaryKey}}
		entity.{{$field.Name}},
		{{- end}}
		{{- end}}
		entity.{{.PrimaryKey}},
	)
	
	return err
}

// Delete 删除实体
func (r *{{.EntityName}}RepositoryImpl) Delete(ctx context.Context, id {{.PrimaryKeyType}}) error {
	query := ` + "`DELETE FROM {{.TableName}} WHERE {{range $field := .Fields}}{{if $field.IsPrimaryKey}}{{$field.ColumnName}}{{end}}{{end}} = ?`" + `
	
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}`

// 服务模板
const serviceTemplate = `// Code generated by gRain. DO NOT EDIT.
package {{.PackageName}}

import (
	"context"
	{{range .Imports}}
	"{{.}}"
	{{end}}
	"github.com/isBlue-5/grain/pkg/annotation/types"
)

// {{.EntityName}}Service 定义{{.EntityName}}服务接口
// frame:service
type {{.EntityName}}Service interface {
	GetByID(ctx context.Context, id {{.PrimaryKeyType}}) (*{{.EntityName}}, error)
	GetAll(ctx context.Context) ([]*{{.EntityName}}, error)
	Create(ctx context.Context, entity *{{.EntityName}}) error
	Update(ctx context.Context, entity *{{.EntityName}}) error
	Delete(ctx context.Context, id {{.PrimaryKeyType}}) error
}

// {{.EntityName}}ServiceImpl 实现{{.EntityName}}服务接口
type {{.EntityName}}ServiceImpl struct {
	repository {{.EntityName}}Repository
}

// New{{.EntityName}}Service 创建{{.EntityName}}服务实现
// frame:inject
func New{{.EntityName}}Service(repo {{.EntityName}}Repository) {{.EntityName}}Service {
	return &{{.EntityName}}ServiceImpl{
		repository: repo,
	}
}

// GetByID 根据ID获取{{.EntityName}}
// frame:transaction(readOnly=true)
// frame:log(level="INFO")
func (s *{{.EntityName}}ServiceImpl) GetByID(ctx context.Context, id {{.PrimaryKeyType}}) (*{{.EntityName}}, error) {
	return s.repository.FindByID(ctx, id)
}

// GetAll 获取所有{{.EntityName}}
// frame:transaction(readOnly=true)
// frame:log(level="INFO")
// frame:cache(key="all_{{.EntityName}}s", ttl="5m")
func (s *{{.EntityName}}ServiceImpl) GetAll(ctx context.Context) ([]*{{.EntityName}}, error) {
	return s.repository.FindAll(ctx)
}

// Create 创建{{.EntityName}}
// frame:transaction
// frame:log(level="INFO")
func (s *{{.EntityName}}ServiceImpl) Create(ctx context.Context, entity *{{.EntityName}}) error {
	// 可以在这里添加业务验证逻辑
	return s.repository.Save(ctx, entity)
}

// Update 更新{{.EntityName}}
// frame:transaction
// frame:log(level="INFO")
func (s *{{.EntityName}}ServiceImpl) Update(ctx context.Context, entity *{{.EntityName}}) error {
	// 可以在这里添加业务验证逻辑
	return s.repository.Update(ctx, entity)
}

// Delete 删除{{.EntityName}}
// frame:transaction
// frame:log(level="INFO")
func (s *{{.EntityName}}ServiceImpl) Delete(ctx context.Context, id {{.PrimaryKeyType}}) error {
	return s.repository.Delete(ctx, id)
}
`

// 控制器模板
const controllerTemplate = `// Code generated by gRain. DO NOT EDIT.
package {{.PackageName}}

import (
	"net/http"
	"strconv"
	{{range .Imports}}
	"{{.}}"
	{{end}}
	"github.com/gin-gonic/gin"
)

// {{.EntityName}}Controller {{.EntityName}}控制器
// frame:controller(path="/api/{{.EntityName | ToLower}}s")
type {{.EntityName}}Controller struct {
	service {{.EntityName}}Service
}

// New{{.EntityName}}Controller 创建{{.EntityName}}控制器
// frame:inject
func New{{.EntityName}}Controller(service {{.EntityName}}Service) *{{.EntityName}}Controller {
	return &{{.EntityName}}Controller{
		service: service,
	}
}

// GetByID 获取单个实体
// frame:route(method="GET", path="/:id")
// frame:log(level="INFO")
func (c *{{.EntityName}}Controller) GetByID(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	entity, err := c.service.GetByID(ctx, {{.PrimaryKeyType}}(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Entity not found"})
		return
	}

	ctx.JSON(http.StatusOK, entity)
}

// GetAll 获取所有实体
// frame:route(method="GET", path="/")
// frame:log(level="INFO")
func (c *{{.EntityName}}Controller) GetAll(ctx *gin.Context) {
	entities, err := c.service.GetAll(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, entities)
}

// Create 创建新实体
// frame:route(method="POST", path="/")
// frame:log(level="INFO")
func (c *{{.EntityName}}Controller) Create(ctx *gin.Context) {
	var entity {{.EntityName}}
	if err := ctx.ShouldBindJSON(&entity); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.service.Create(ctx, &entity); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, entity)
}

// Update 更新实体
// frame:route(method="PUT", path="/:id")
// frame:log(level="INFO")
func (c *{{.EntityName}}Controller) Update(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var entity {{.EntityName}}
	if err := ctx.ShouldBindJSON(&entity); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 设置ID
	entity.ID = {{.PrimaryKeyType}}(id)

	if err := c.service.Update(ctx, &entity); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, entity)
}

// Delete 删除实体
// frame:route(method="DELETE", path="/:id")
// frame:log(level="INFO")
func (c *{{.EntityName}}Controller) Delete(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	if err := c.service.Delete(ctx, {{.PrimaryKeyType}}(id)); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Entity deleted successfully"})
}
`
