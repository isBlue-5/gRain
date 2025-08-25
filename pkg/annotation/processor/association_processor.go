// Package processor 提供注解处理器实现，用于生成代码
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

// AssociationProcessor 关联关系处理器，用于处理实体之间的关联关系
type AssociationProcessor struct {
	registry   registry.Registry
	parser     AnnotationParser
	outputPath string
	tmpl       *template.Template
	verbose    bool
}

// 关联注解类型常量
const (
	OneToOneType   = "oneToOne"   // 一对一关系
	OneToManyType  = "oneToMany"  // 一对多关系
	ManyToOneType  = "manyToOne"  // 多对一关系
	ManyToManyType = "manyToMany" // 多对多关系
)

// 关联关系模板
const associationTmpl = `
// 关联关系代码
// 关联类型: {{.RelationType}}
// 源实体: {{.SourceEntity}}
// 目标实体: {{.TargetEntity}}
// 关联字段: {{.FieldName}}

{{if eq .RelationType "oneToOne"}}
// 一对一关系处理方法
func (r *{{.RepositoryName}}) Find{{.TargetEntity}}By{{.SourceEntity}}ID(ctx context.Context, {{.SourceIdParam}} {{.SourceIdType}}) (*{{.TargetEntity}}, error) {
	var result {{.TargetEntity}}
	err := r.db.WithContext(ctx).
		Joins("JOIN {{.JoinTable}} ON {{.JoinTable}}.{{.JoinForeignKey}} = {{.TargetTable}}.{{.TargetPrimaryKey}}").
		Where("{{.JoinTable}}.{{.JoinReferences}} = ?", {{.SourceIdParam}}).
		First(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

{{else if eq .RelationType "oneToMany"}}
// 一对多关系处理方法
func (r *{{.RepositoryName}}) Find{{.TargetEntity}}sBy{{.SourceEntity}}ID(ctx context.Context, {{.SourceIdParam}} {{.SourceIdType}}) ([]*{{.TargetEntity}}, error) {
	var results []*{{.TargetEntity}}
	err := r.db.WithContext(ctx).
		Where("{{.ForeignKey}} = ?", {{.SourceIdParam}}).
		Find(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (r *{{.RepositoryName}}) Add{{.TargetEntity}}To{{.SourceEntity}}(ctx context.Context, {{.SourceIdParam}} {{.SourceIdType}}, {{.TargetIdParam}} {{.TargetIdType}}) error {
	// 查找源实体
	source{{.SourceEntity}}, err := r.FindByID(ctx, {{.SourceIdParam}})
	if err != nil {
		return err
	}
	
	// 查找目标实体
	target{{.TargetEntity}}, err := r.{{.TargetEntity}}Repository.FindByID(ctx, {{.TargetIdParam}})
	if err != nil {
		return err
	}
	
	// 设置关联关系
	target{{.TargetEntity}}.{{.ForeignKey}} = {{.SourceIdParam}}
	
	// 更新目标实体
	return r.{{.TargetEntity}}Repository.Update(ctx, target{{.TargetEntity}})
}

func (r *{{.RepositoryName}}) Remove{{.TargetEntity}}From{{.SourceEntity}}(ctx context.Context, {{.SourceIdParam}} {{.SourceIdType}}, {{.TargetIdParam}} {{.TargetIdType}}) error {
	// 查找目标实体
	target{{.TargetEntity}}, err := r.{{.TargetEntity}}Repository.FindByID(ctx, {{.TargetIdParam}})
	if err != nil {
		return err
	}
	
	// 检查关联关系
	if target{{.TargetEntity}}.{{.ForeignKey}} != {{.SourceIdParam}} {
		return fmt.Errorf("{{.TargetEntity}} with ID %v is not associated with {{.SourceEntity}} with ID %v", {{.TargetIdParam}}, {{.SourceIdParam}})
	}
	
	// 清除关联关系
	target{{.TargetEntity}}.{{.ForeignKey}} = 0 // 假设ID类型为数字
	
	// 更新目标实体
	return r.{{.TargetEntity}}Repository.Update(ctx, target{{.TargetEntity}})
}

{{else if eq .RelationType "manyToOne"}}
// 多对一关系处理方法
func (r *{{.RepositoryName}}) Find{{.TargetEntity}}By{{.SourceEntity}}ID(ctx context.Context, {{.SourceIdParam}} {{.SourceIdType}}) (*{{.TargetEntity}}, error) {
	// 查找源实体
	source{{.SourceEntity}}, err := r.FindByID(ctx, {{.SourceIdParam}})
	if err != nil {
		return nil, err
	}
	
	// 查找目标实体
	return r.{{.TargetEntity}}Repository.FindByID(ctx, source{{.SourceEntity}}.{{.ForeignKey}})
}

func (r *{{.RepositoryName}}) Set{{.TargetEntity}}For{{.SourceEntity}}(ctx context.Context, {{.SourceIdParam}} {{.SourceIdType}}, {{.TargetIdParam}} {{.TargetIdType}}) error {
	// 查找源实体
	source{{.SourceEntity}}, err := r.FindByID(ctx, {{.SourceIdParam}})
	if err != nil {
		return err
	}
	
	// 查找目标实体
	_, err = r.{{.TargetEntity}}Repository.FindByID(ctx, {{.TargetIdParam}})
	if err != nil {
		return err
	}
	
	// 设置关联关系
	source{{.SourceEntity}}.{{.ForeignKey}} = {{.TargetIdParam}}
	
	// 更新源实体
	return r.Update(ctx, source{{.SourceEntity}})
}

{{else if eq .RelationType "manyToMany"}}
// 多对多关系处理方法
func (r *{{.RepositoryName}}) Find{{.TargetEntity}}sBy{{.SourceEntity}}ID(ctx context.Context, {{.SourceIdParam}} {{.SourceIdType}}) ([]*{{.TargetEntity}}, error) {
	var results []*{{.TargetEntity}}
	err := r.db.WithContext(ctx).
		Joins("JOIN {{.JoinTable}} ON {{.JoinTable}}.{{.JoinForeignKey}} = {{.TargetTable}}.{{.TargetPrimaryKey}}").
		Where("{{.JoinTable}}.{{.JoinReferences}} = ?", {{.SourceIdParam}}).
		Find(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (r *{{.RepositoryName}}) Find{{.SourceEntity}}sBy{{.TargetEntity}}ID(ctx context.Context, {{.TargetIdParam}} {{.TargetIdType}}) ([]*{{.SourceEntity}}, error) {
	var results []*{{.SourceEntity}}
	err := r.db.WithContext(ctx).
		Joins("JOIN {{.JoinTable}} ON {{.JoinTable}}.{{.JoinReferences}} = {{.SourceTable}}.{{.SourcePrimaryKey}}").
		Where("{{.JoinTable}}.{{.JoinForeignKey}} = ?", {{.TargetIdParam}}).
		Find(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (r *{{.RepositoryName}}) Add{{.TargetEntity}}To{{.SourceEntity}}(ctx context.Context, {{.SourceIdParam}} {{.SourceIdType}}, {{.TargetIdParam}} {{.TargetIdType}}) error {
	// 创建关联记录
	result := r.db.WithContext(ctx).Exec(
		"INSERT INTO {{.JoinTable}} ({{.JoinReferences}}, {{.JoinForeignKey}}) VALUES (?, ?)",
		{{.SourceIdParam}}, {{.TargetIdParam}},
	)
	return result.Error
}

func (r *{{.RepositoryName}}) Remove{{.TargetEntity}}From{{.SourceEntity}}(ctx context.Context, {{.SourceIdParam}} {{.SourceIdType}}, {{.TargetIdParam}} {{.TargetIdType}}) error {
	// 删除关联记录
	result := r.db.WithContext(ctx).Exec(
		"DELETE FROM {{.JoinTable}} WHERE {{.JoinReferences}} = ? AND {{.JoinForeignKey}} = ?",
		{{.SourceIdParam}}, {{.TargetIdParam}},
	)
	return result.Error
}

func (r *{{.RepositoryName}}) RemoveAll{{.TargetEntity}}sFrom{{.SourceEntity}}(ctx context.Context, {{.SourceIdParam}} {{.SourceIdType}}) error {
	// 删除所有关联记录
	result := r.db.WithContext(ctx).Exec(
		"DELETE FROM {{.JoinTable}} WHERE {{.JoinReferences}} = ?",
		{{.SourceIdParam}},
	)
	return result.Error
}
{{end}}
`

// AssociationInfo 关联关系信息
type AssociationInfo struct {
	RelationType     string // 关联类型
	SourceEntity     string // 源实体
	TargetEntity     string // 目标实体
	FieldName        string // 关联字段名
	RepositoryName   string // 仓库名称
	SourceTable      string // 源表名
	TargetTable      string // 目标表名
	SourcePrimaryKey string // 源主键
	TargetPrimaryKey string // 目标主键
	ForeignKey       string // 外键
	References       string // 引用字段
	JoinTable        string // 中间表
	JoinForeignKey   string // 中间表外键
	JoinReferences   string // 中间表引用字段
	SourceIdParam    string // 源ID参数名
	SourceIdType     string // 源ID类型
	TargetIdParam    string // 目标ID参数名
	TargetIdType     string // 目标ID类型
	FetchType        string // 加载类型
	Cascade          string // 级联操作
}

// NewAssociationProcessor 创建关联关系处理器
func NewAssociationProcessor(reg registry.Registry, parser AnnotationParser, outputPath string) *AssociationProcessor {
	tmpl, err := template.New("association").Parse(associationTmpl)
	if err != nil {
		panic(fmt.Errorf("解析关联关系模板失败: %w", err))
	}

	return &AssociationProcessor{
		registry:   reg,
		parser:     parser,
		outputPath: outputPath,
		tmpl:       tmpl,
		verbose:    false,
	}
}

// WithVerbose 设置是否启用详细日志
func (p *AssociationProcessor) WithVerbose(verbose bool) *AssociationProcessor {
	p.verbose = verbose
	return p
}

// ProcessAssociation 处理关联关系注解
func (p *AssociationProcessor) ProcessAssociation(paths []string) error {
	// 获取所有实体注解
	var entityAnnotations []types.Annotation

	// 解析所有路径下的注解
	for _, path := range paths {
		anns, err := p.parser.ParseDir(path)
		if err != nil {
			return fmt.Errorf("解析目录 %s 失败: %w", path, err)
		}

		for _, ann := range anns {
			if ann.GetType() == types.EntityType {
				entityAnnotations = append(entityAnnotations, ann)
			}
		}
	}

	if p.verbose {
		fmt.Printf("找到 %d 个实体注解\n", len(entityAnnotations))
	}

	// 处理每个实体的关联关系
	for _, ann := range entityAnnotations {
		if err := p.processEntityAssociations(ann); err != nil {
			return err
		}
	}

	return nil
}

// processEntityAssociations 处理实体的关联关系
func (p *AssociationProcessor) processEntityAssociations(ann types.Annotation) error {
	// 获取实体结构体
	var structType *ast.StructType
	var structName string

	if structAnn, ok := ann.(*types.StructTagAnnotation); ok {
		structType = structAnn.StructType
		structName = structAnn.StructName
	} else if commentAnn, ok := ann.(*types.CommentAnnotation); ok {
		if typeSpec, ok := commentAnn.Node.(*ast.TypeSpec); ok {
			if st, ok := typeSpec.Type.(*ast.StructType); ok {
				structType = st
				structName = typeSpec.Name.Name
			}
		}
	}

	if structType == nil {
		return fmt.Errorf("无法获取实体结构体: %s", ann.GetTargetName())
	}

	// 获取表名
	tableName := types.GetStringAttribute(ann.GetAttributes(), "table", strings.ToLower(structName)+"s")

	// 获取主键
	primaryKey := "ID"
	primaryKeyType := "uint"

	// 处理结构体字段
	for _, field := range structType.Fields.List {
		// 检查字段是否有关联注解
		var assocType string
		var targetEntity string

		// 检查字段标签
		if field.Tag != nil {
			tag := strings.Trim(field.Tag.Value, "`")
			if strings.Contains(tag, "frame:oneToOne") {
				assocType = OneToOneType
			} else if strings.Contains(tag, "frame:oneToMany") {
				assocType = OneToManyType
			} else if strings.Contains(tag, "frame:manyToOne") {
				assocType = ManyToOneType
			} else if strings.Contains(tag, "frame:manyToMany") {
				assocType = ManyToManyType
			}

			// 提取目标实体
			if assocType != "" {
				targetEntity = extractAssociationTagValue(tag, "targetEntity")
				if targetEntity == "" {
					// 从字段类型推断目标实体
					targetEntity = extractFieldType(field)
				}
			}
		}

		// 检查字段注释
		if field.Doc != nil {
			for _, doc := range field.Doc.List {
				text := doc.Text
				if strings.Contains(text, "frame:oneToOne") {
					assocType = OneToOneType
				} else if strings.Contains(text, "frame:oneToMany") {
					assocType = OneToManyType
				} else if strings.Contains(text, "frame:manyToOne") {
					assocType = ManyToOneType
				} else if strings.Contains(text, "frame:manyToMany") {
					assocType = ManyToManyType
				}

				// 提取目标实体
				if assocType != "" && targetEntity == "" {
					targetEntity = extractCommentValue(text, "targetEntity")
					if targetEntity == "" {
						// 从字段类型推断目标实体
						targetEntity = extractFieldType(field)
					}
				}
			}
		}

		// 处理关联关系
		if assocType != "" && targetEntity != "" {
			// 获取字段名
			fieldName := ""
			if len(field.Names) > 0 {
				fieldName = field.Names[0].Name
			}

			// 创建关联信息
			assocInfo := &AssociationInfo{
				RelationType:     assocType,
				SourceEntity:     structName,
				TargetEntity:     targetEntity,
				FieldName:        fieldName,
				RepositoryName:   structName + "Repository",
				SourceTable:      tableName,
				TargetTable:      strings.ToLower(targetEntity) + "s",
				SourcePrimaryKey: primaryKey,
				TargetPrimaryKey: primaryKey,
				SourceIdParam:    strings.ToLower(structName) + "ID",
				SourceIdType:     primaryKeyType,
				TargetIdParam:    strings.ToLower(targetEntity) + "ID",
				TargetIdType:     primaryKeyType,
			}

			// 设置关联属性
			switch assocType {
			case OneToOneType:
				assocInfo.ForeignKey = strings.ToLower(structName) + "_id"
				assocInfo.References = primaryKey

			case OneToManyType:
				assocInfo.ForeignKey = strings.ToLower(structName) + "_id"
				assocInfo.References = primaryKey

			case ManyToOneType:
				assocInfo.ForeignKey = strings.ToLower(targetEntity) + "_id"
				assocInfo.References = primaryKey

			case ManyToManyType:
				assocInfo.JoinTable = strings.ToLower(structName) + "_" + strings.ToLower(targetEntity)
				assocInfo.JoinForeignKey = strings.ToLower(targetEntity) + "_id"
				assocInfo.JoinReferences = strings.ToLower(structName) + "_id"
			}

			// 生成关联代码
			if err := p.generateAssociationCode(assocInfo); err != nil {
				return err
			}
		}
	}

	return nil
}

// generateAssociationCode 生成关联关系代码
func (p *AssociationProcessor) generateAssociationCode(assocInfo *AssociationInfo) error {
	// 创建输出文件
	outputFile := filepath.Join(p.outputPath, fmt.Sprintf("%s_%s_association.go",
		strings.ToLower(assocInfo.SourceEntity),
		strings.ToLower(assocInfo.TargetEntity)))

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
	if err := p.tmpl.Execute(file, assocInfo); err != nil {
		return fmt.Errorf("生成关联代码失败: %w", err)
	}

	return nil
}

// extractFieldType 从字段提取类型名称
func extractFieldType(field *ast.Field) string {
	switch t := field.Type.(type) {
	case *ast.Ident:
		// 直接类型，如 User
		return t.Name
	case *ast.StarExpr:
		// 指针类型，如 *User
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	case *ast.ArrayType:
		// 数组类型，如 []User 或 []*User
		if starExpr, ok := t.Elt.(*ast.StarExpr); ok {
			if ident, ok := starExpr.X.(*ast.Ident); ok {
				return ident.Name
			}
		} else if ident, ok := t.Elt.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}

// extractAssociationTagValue 从标签中提取关联值
func extractAssociationTagValue(tag, key string) string {
	prefix := key + "="
	parts := strings.Split(tag, " ")
	for _, part := range parts {
		if strings.HasPrefix(part, prefix) {
			value := part[len(prefix):]
			return strings.Trim(value, "\"")
		}
	}
	return ""
}

// extractCommentValue 从注释中提取值
func extractCommentValue(comment, key string) string {
	prefix := key + "="
	start := strings.Index(comment, prefix)
	if start == -1 {
		return ""
	}

	start += len(prefix)
	end := strings.IndexAny(comment[start:], " ,)")
	if end == -1 {
		return strings.Trim(comment[start:], "\"")
	}

	return strings.Trim(comment[start:start+end], "\"")
}
