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

// QueryProcessor 查询注解处理器，用于处理自定义查询注解
type QueryProcessor struct {
	registry   registry.Registry
	parser     AnnotationParser
	outputPath string
	tmpl       *template.Template
	verbose    bool
}

// 查询方法类型
const (
	FindByType    = "findBy"    // 查询单个实体
	FindAllByType = "findAllBy" // 查询多个实体
	CountByType   = "countBy"   // 计数查询
	ExistsByType  = "existsBy"  // 存在性查询
	DeleteByType  = "deleteBy"  // 删除查询
	UpdateByType  = "updateBy"  // 更新查询
)

// 查询条件操作符
const (
	Equals              = "Equals"              // 等于
	NotEquals           = "NotEquals"           // 不等于
	GreaterThan         = "GreaterThan"         // 大于
	GreaterThanEqual    = "GreaterThanEqual"    // 大于等于
	LessThan            = "LessThan"            // 小于
	LessThanEqual       = "LessThanEqual"       // 小于等于
	Like                = "Like"                // 模糊匹配
	NotLike             = "NotLike"             // 不匹配
	In                  = "In"                  // 包含于集合
	NotIn               = "NotIn"               // 不包含于集合
	IsNull              = "IsNull"              // 为空
	IsNotNull           = "IsNotNull"           // 不为空
	Between             = "Between"             // 范围查询
	OrderBy             = "OrderBy"             // 排序
	OrderByDesc         = "OrderByDesc"         // 降序排序
	Limit               = "Limit"               // 限制结果数量
	Offset              = "Offset"              // 结果偏移量
	And                 = "And"                 // 逻辑与
	Or                  = "Or"                  // 逻辑或
	StartingWith        = "StartingWith"        // 前缀匹配
	EndingWith          = "EndingWith"          // 后缀匹配
	Containing          = "Containing"          // 包含
	NotContaining       = "NotContaining"       // 不包含
	IgnoreCase          = "IgnoreCase"          // 忽略大小写
	True                = "True"                // 为真
	False               = "False"               // 为假
	After               = "After"               // 时间在之后
	Before              = "Before"              // 时间在之前
	FirstResult         = "FirstResult"         // 第一个结果
	MaxResults          = "MaxResults"          // 最大结果数
	DistinctBy          = "DistinctBy"          // 去重
	GroupBy             = "GroupBy"             // 分组
	Having              = "Having"              // 分组条件
	Join                = "Join"                // 联表
	LeftJoin            = "LeftJoin"            // 左联表
	RightJoin           = "RightJoin"           // 右联表
	InnerJoin           = "InnerJoin"           // 内联表
	WithLock            = "WithLock"            // 加锁
	WithPessimisticLock = "WithPessimisticLock" // 悲观锁
	WithOptimisticLock  = "WithOptimisticLock"  // 乐观锁
)

// 查询方法模板
const queryMethodTmpl = `
// {{.MethodName}} {{.MethodComment}}
{{- if .IsInterface}}
func (r {{.ReceiverType}}) {{.MethodName}}({{.Params}}) ({{.Returns}})
{{- else}}
func (r {{.ReceiverType}}) {{.MethodName}}({{.Params}}) ({{.Returns}}) {
	{{- if .HasContext}}
	db := r.db.WithContext({{.ContextName}})
	{{- else}}
	db := r.db
	{{- end}}
	
	{{- if eq .QueryType "findBy"}}
	var result {{.EntityType}}
	query := {{.BuildQueryString}}
	err := db{{.BuildQueryChain}}.First(&result).Error
	if err != nil {
		{{- if eq .ErrorHandling "nil"}}
		return nil, nil
		{{- else if eq .ErrorHandling "error"}}
		return nil, err
		{{- else if eq .ErrorHandling "gorm.ErrRecordNotFound"}}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
		{{- else}}
		return nil, err
		{{- end}}
	}
	return &result, nil
	
	{{- else if eq .QueryType "findAllBy"}}
	var results []*{{.EntityType}}
	query := {{.BuildQueryString}}
	err := db{{.BuildQueryChain}}{{if .Pagination}}.Limit(limit).Offset(offset){{end}}.Find(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
	
	{{- else if eq .QueryType "countBy"}}
	var count int64
	query := {{.BuildQueryString}}
	err := db{{.BuildQueryChain}}.Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
	
	{{- else if eq .QueryType "existsBy"}}
	var count int64
	query := {{.BuildQueryString}}
	err := db{{.BuildQueryChain}}.Limit(1).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
	
	{{- else if eq .QueryType "deleteBy"}}
	query := {{.BuildQueryString}}
	result := db{{.BuildQueryChain}}.Delete(&{{.EntityType}}{})
	return result.RowsAffected, result.Error
	
	{{- else if eq .QueryType "updateBy"}}
	query := {{.BuildQueryString}}
	result := db{{.BuildQueryChain}}.Updates(updates)
	return result.RowsAffected, result.Error
	{{- end}}
}
{{- end}}
`

// QueryMethodInfo 查询方法信息
type QueryMethodInfo struct {
	MethodName       string   // 方法名
	MethodComment    string   // 方法注释
	ReceiverType     string   // 接收者类型
	Params           string   // 参数列表
	Returns          string   // 返回值列表
	EntityType       string   // 实体类型
	QueryType        string   // 查询类型
	QueryFields      []string // 查询字段
	QueryOperators   []string // 查询操作符
	BuildQueryString string   // 构建查询语句
	BuildQueryChain  string   // 构建查询链
	HasContext       bool     // 是否有上下文参数
	ContextName      string   // 上下文参数名
	Pagination       bool     // 是否支持分页
	ErrorHandling    string   // 错误处理策略
	IsInterface      bool     // 是否是接口方法
}

// NewQueryProcessor 创建查询处理器
func NewQueryProcessor(reg registry.Registry, parser AnnotationParser, outputPath string) *QueryProcessor {
	tmpl, err := template.New("query").Parse(queryMethodTmpl)
	if err != nil {
		panic(fmt.Errorf("解析查询模板失败: %w", err))
	}

	return &QueryProcessor{
		registry:   reg,
		parser:     parser,
		outputPath: outputPath,
		tmpl:       tmpl,
		verbose:    false,
	}
}

// WithVerbose 设置是否启用详细日志
func (p *QueryProcessor) WithVerbose(verbose bool) *QueryProcessor {
	p.verbose = verbose
	return p
}

// ProcessQuery 处理查询注解
func (p *QueryProcessor) ProcessQuery(paths []string) error {
	// 获取所有查询注解
	var annotations []types.Annotation

	// 解析所有路径下的注解
	for _, path := range paths {
		anns, err := p.parser.ParseDir(path)
		if err != nil {
			return fmt.Errorf("解析目录 %s 失败: %w", path, err)
		}

		for _, ann := range anns {
			if ann.GetType() == types.QueryType {
				annotations = append(annotations, ann)
			}
		}
	}

	if p.verbose {
		fmt.Printf("找到 %d 个查询注解\n", len(annotations))
	}

	// 按包进行分组处理
	packageMap := make(map[string][]types.CommentAnnotation)
	for _, ann := range annotations {
		if commentAnn, ok := ann.(*types.CommentAnnotation); ok {
			pkgName := filepath.Base(filepath.Dir(commentAnn.Position.Filename))
			packageMap[pkgName] = append(packageMap[pkgName], *commentAnn)
		}
	}

	// 处理每个包的查询注解
	for pkgName, anns := range packageMap {
		if err := p.processPackage(pkgName, anns); err != nil {
			return err
		}
	}

	return nil
}

// processPackage 处理单个包的查询注解
func (p *QueryProcessor) processPackage(pkgName string, annotations []types.CommentAnnotation) error {
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

// processMethodAnnotation 处理方法上的查询注解
func (p *QueryProcessor) processMethodAnnotation(pkgName string, ann types.CommentAnnotation) error {
	// 提取注解参数
	entityType := types.GetStringAttribute(ann.Attributes, "entity", "")
	if entityType == "" {
		return fmt.Errorf("查询注解必须指定entity属性: %s", ann.TargetName)
	}

	queryType := types.GetStringAttribute(ann.Attributes, "type", "findBy")
	errorHandling := types.GetStringAttribute(ann.Attributes, "errorHandling", "error")
	pagination := types.GetBoolAttribute(ann.Attributes, "pagination", false)

	// 分析方法
	methodDecl, ok := ann.Node.(*ast.FuncDecl)
	if !ok {
		return fmt.Errorf("查询注解只能应用于方法或函数: %s", ann.TargetName)
	}

	// 提取方法签名信息
	signatureInfo, err := extractMethodSignature(methodDecl)
	if err != nil {
		return fmt.Errorf("提取方法签名失败: %w", err)
	}

	// 解析方法名以提取查询字段和操作符
	methodName := methodDecl.Name.Name
	queryFields, queryOperators := parseMethodName(methodName, queryType)

	// 构建查询语句
	buildQueryString, buildQueryChain := buildQueryExpression(queryFields, queryOperators, entityType)

	// 获取方法注释
	methodComment := extractQueryDocComment(methodDecl.Doc)
	if methodComment == "" {
		methodComment = fmt.Sprintf("根据%s查询%s", strings.Join(queryFields, "、"), entityType)
	}

	// 创建模板数据
	tmplData := &QueryMethodInfo{
		MethodName:       methodName,
		MethodComment:    methodComment,
		ReceiverType:     signatureInfo["ReceiverType"].(string),
		Params:           signatureInfo["Params"].(string),
		Returns:          signatureInfo["Returns"].(string),
		EntityType:       entityType,
		QueryType:        queryType,
		QueryFields:      queryFields,
		QueryOperators:   queryOperators,
		BuildQueryString: buildQueryString,
		BuildQueryChain:  buildQueryChain,
		HasContext:       signatureInfo["ContextName"].(string) != "",
		ContextName:      signatureInfo["ContextName"].(string),
		Pagination:       pagination,
		ErrorHandling:    errorHandling,
		IsInterface:      types.GetBoolAttribute(ann.Attributes, "interface", false),
	}

	// 生成查询代码文件
	outputFile := filepath.Join(p.outputPath, fmt.Sprintf("%s_query.go", strings.ToLower(ann.TargetName)))

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
		return fmt.Errorf("生成查询代码失败: %w", err)
	}

	return nil
}

// processFunctionAnnotation 处理函数上的查询注解
func (p *QueryProcessor) processFunctionAnnotation(pkgName string, ann types.CommentAnnotation) error {
	// 函数级查询处理逻辑（类似方法处理，但无接收者）
	return nil
}

// parseMethodName 解析方法名以提取查询字段和操作符
func parseMethodName(methodName string, queryType string) ([]string, []string) {
	// 去掉方法名前缀
	name := methodName
	if strings.HasPrefix(name, "FindBy") {
		name = name[6:]
	} else if strings.HasPrefix(name, "FindAllBy") {
		name = name[9:]
	} else if strings.HasPrefix(name, "CountBy") {
		name = name[7:]
	} else if strings.HasPrefix(name, "ExistsBy") {
		name = name[8:]
	} else if strings.HasPrefix(name, "DeleteBy") {
		name = name[8:]
	} else if strings.HasPrefix(name, "UpdateBy") {
		name = name[8:]
	}

	// 分割字段和操作符
	parts := splitCamelCase(name)
	fields := make([]string, 0)
	operators := make([]string, 0)

	currentField := ""
	for i := 0; i < len(parts); i++ {
		part := parts[i]

		// 检查是否是操作符
		isOperator := false
		for _, op := range getAllOperators() {
			if part == op {
				if currentField != "" {
					fields = append(fields, currentField)
					currentField = ""
				}
				operators = append(operators, part)
				isOperator = true
				break
			}
		}

		if !isOperator {
			if currentField == "" {
				currentField = part
			} else {
				currentField = currentField + part
			}
		}
	}

	// 添加最后一个字段
	if currentField != "" {
		fields = append(fields, currentField)
	}

	// 如果没有操作符，默认为Equals
	if len(operators) == 0 {
		operators = append(operators, "Equals")
	}

	return fields, operators
}

// splitCamelCase 将驼峰命名分割为单词
func splitCamelCase(s string) []string {
	var words []string
	var currentWord string

	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			// 遇到大写字母，结束当前单词
			if currentWord != "" {
				words = append(words, currentWord)
				currentWord = ""
			}
			currentWord = string(r)
		} else {
			currentWord += string(r)
		}
	}

	// 添加最后一个单词
	if currentWord != "" {
		words = append(words, currentWord)
	}

	return words
}

// getAllOperators 获取所有操作符
func getAllOperators() []string {
	return []string{
		Equals, NotEquals, GreaterThan, GreaterThanEqual, LessThan, LessThanEqual,
		Like, NotLike, In, NotIn, IsNull, IsNotNull, Between, OrderBy, OrderByDesc,
		Limit, Offset, And, Or, StartingWith, EndingWith, Containing, NotContaining,
		IgnoreCase, True, False, After, Before, FirstResult, MaxResults, DistinctBy,
		GroupBy, Having, Join, LeftJoin, RightJoin, InnerJoin, WithLock,
		WithPessimisticLock, WithOptimisticLock,
	}
}

// buildQueryExpression 构建查询表达式
func buildQueryExpression(fields []string, operators []string, entityType string) (string, string) {
	// 简单实现，实际应根据字段和操作符生成复杂的查询表达式
	conditions := make([]string, 0)
	chains := make([]string, 0)

	for i, field := range fields {
		op := "Equals"
		if i < len(operators) {
			op = operators[i]
		}

		// 转换字段名首字母为小写
		fieldName := strings.ToLower(field[:1]) + field[1:]

		switch op {
		case Equals:
			conditions = append(conditions, fmt.Sprintf("%s = ?", fieldName))
			chains = append(chains, fmt.Sprintf(".Where(\"%s = ?\", %s)", fieldName, fieldName))
		case NotEquals:
			conditions = append(conditions, fmt.Sprintf("%s <> ?", fieldName))
			chains = append(chains, fmt.Sprintf(".Where(\"%s <> ?\", %s)", fieldName, fieldName))
		case GreaterThan:
			conditions = append(conditions, fmt.Sprintf("%s > ?", fieldName))
			chains = append(chains, fmt.Sprintf(".Where(\"%s > ?\", %s)", fieldName, fieldName))
		case GreaterThanEqual:
			conditions = append(conditions, fmt.Sprintf("%s >= ?", fieldName))
			chains = append(chains, fmt.Sprintf(".Where(\"%s >= ?\", %s)", fieldName, fieldName))
		case LessThan:
			conditions = append(conditions, fmt.Sprintf("%s < ?", fieldName))
			chains = append(chains, fmt.Sprintf(".Where(\"%s < ?\", %s)", fieldName, fieldName))
		case LessThanEqual:
			conditions = append(conditions, fmt.Sprintf("%s <= ?", fieldName))
			chains = append(chains, fmt.Sprintf(".Where(\"%s <= ?\", %s)", fieldName, fieldName))
		case Like:
			conditions = append(conditions, fmt.Sprintf("%s LIKE ?", fieldName))
			chains = append(chains, fmt.Sprintf(".Where(\"%s LIKE ?\", \"%%\"+%s+\"%%\")", fieldName, fieldName))
		case NotLike:
			conditions = append(conditions, fmt.Sprintf("%s NOT LIKE ?", fieldName))
			chains = append(chains, fmt.Sprintf(".Where(\"%s NOT LIKE ?\", \"%%\"+%s+\"%%\")", fieldName, fieldName))
		case In:
			conditions = append(conditions, fmt.Sprintf("%s IN ?", fieldName))
			chains = append(chains, fmt.Sprintf(".Where(\"%s IN ?\", %s)", fieldName, fieldName))
		case NotIn:
			conditions = append(conditions, fmt.Sprintf("%s NOT IN ?", fieldName))
			chains = append(chains, fmt.Sprintf(".Where(\"%s NOT IN ?\", %s)", fieldName, fieldName))
		case IsNull:
			conditions = append(conditions, fmt.Sprintf("%s IS NULL", fieldName))
			chains = append(chains, fmt.Sprintf(".Where(\"%s IS NULL\")", fieldName))
		case IsNotNull:
			conditions = append(conditions, fmt.Sprintf("%s IS NOT NULL", fieldName))
			chains = append(chains, fmt.Sprintf(".Where(\"%s IS NOT NULL\")", fieldName))
		case Between:
			conditions = append(conditions, fmt.Sprintf("%s BETWEEN ? AND ?", fieldName))
			chains = append(chains, fmt.Sprintf(".Where(\"%s BETWEEN ? AND ?\", %sStart, %sEnd)", fieldName, fieldName, fieldName))
		case OrderBy:
			chains = append(chains, fmt.Sprintf(".Order(\"%s ASC\")", fieldName))
		case OrderByDesc:
			chains = append(chains, fmt.Sprintf(".Order(\"%s DESC\")", fieldName))
		case StartingWith:
			conditions = append(conditions, fmt.Sprintf("%s LIKE ?", fieldName))
			chains = append(chains, fmt.Sprintf(".Where(\"%s LIKE ?\", %s+\"%%\")", fieldName, fieldName))
		case EndingWith:
			conditions = append(conditions, fmt.Sprintf("%s LIKE ?", fieldName))
			chains = append(chains, fmt.Sprintf(".Where(\"%s LIKE ?\", \"%%\"+%s)", fieldName, fieldName))
		case Containing:
			conditions = append(conditions, fmt.Sprintf("%s LIKE ?", fieldName))
			chains = append(chains, fmt.Sprintf(".Where(\"%s LIKE ?\", \"%%\"+%s+\"%%\")", fieldName, fieldName))
		case NotContaining:
			conditions = append(conditions, fmt.Sprintf("%s NOT LIKE ?", fieldName))
			chains = append(chains, fmt.Sprintf(".Where(\"%s NOT LIKE ?\", \"%%\"+%s+\"%%\")", fieldName, fieldName))
		}
	}

	queryString := fmt.Sprintf("\"%s\"", strings.Join(conditions, " AND "))
	queryChain := strings.Join(chains, "")

	return queryString, queryChain
}

// extractQueryDocComment 提取查询文档注释
func extractQueryDocComment(doc *ast.CommentGroup) string {
	if doc == nil {
		return ""
	}

	var comments []string
	for _, comment := range doc.List {
		text := comment.Text
		// 去掉注释前缀
		if strings.HasPrefix(text, "//") {
			text = strings.TrimSpace(text[2:])
		} else if strings.HasPrefix(text, "/*") && strings.HasSuffix(text, "*/") {
			text = strings.TrimSpace(text[2 : len(text)-2])
		}
		comments = append(comments, text)
	}

	return strings.Join(comments, " ")
}
