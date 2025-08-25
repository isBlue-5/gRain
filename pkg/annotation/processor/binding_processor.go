package processor

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"

	"github.com/isBlue-5/grain/pkg/annotation/registry"
	anntypes "github.com/isBlue-5/grain/pkg/annotation/types"
)

// BindingProcessor 处理绑定注解
type BindingProcessor struct {
	// 注解注册中心
	registry registry.Registry

	// 注解前缀
	prefix string

	// 输出目录
	outputDir string
}

// NewBindingProcessor 创建绑定处理器
func NewBindingProcessor(reg registry.Registry, prefix, outputDir string) *BindingProcessor {
	return &BindingProcessor{
		registry:  reg,
		prefix:    prefix,
		outputDir: outputDir,
	}
}

// ProcessBinding 处理绑定注解
func (p *BindingProcessor) ProcessBinding(pkgPaths []string) error {
	// 查找所有绑定注解
	bindAnnotations := p.registry.FindByType(anntypes.BindType)
	validationAnnotations := p.registry.FindByType(anntypes.ValidateType)

	// 按包路径和控制器组织
	packageControllers := make(map[string]map[string][]BindingMethodInfo)

	// 处理绑定注解
	for _, anno := range bindAnnotations {
		if anno.GetTargetType() != anntypes.MethodTarget {
			continue
		}

		info := p.extractBindingInfo(anno)
		pkgPath := p.resolvePackagePath(anno.GetPosition().Filename)

		// 查找对应的验证注解
		for _, vAnno := range validationAnnotations {
			if vAnno.GetTargetType() == anntypes.MethodTarget &&
				vAnno.GetTargetName() == anno.GetTargetName() &&
				filepath.Dir(vAnno.GetPosition().Filename) == pkgPath {
				info.HasValidation = true
				info.ValidationGroups, _ = anntypes.GetStringSliceAttribute(vAnno.GetAttributes(), "groups")
				info.ValidationRules = anntypes.GetStringAttribute(vAnno.GetAttributes(), "rules", "")
				break
			}
		}

		// 组织到包和控制器映射中
		if _, ok := packageControllers[pkgPath]; !ok {
			packageControllers[pkgPath] = make(map[string][]BindingMethodInfo)
		}

		if _, ok := packageControllers[pkgPath][info.ReceiverType]; !ok {
			packageControllers[pkgPath][info.ReceiverType] = []BindingMethodInfo{}
		}

		packageControllers[pkgPath][info.ReceiverType] = append(
			packageControllers[pkgPath][info.ReceiverType],
			info,
		)
	}

	// 为每个包和控制器生成绑定代码
	for pkgPath, controllers := range packageControllers {
		for controller, methods := range controllers {
			if err := p.generateBindingCode(pkgPath, controller, methods); err != nil {
				return fmt.Errorf("生成绑定代码失败 for %s.%s: %w", pkgPath, controller, err)
			}
		}
	}

	return nil
}

// BindingMethodInfo 绑定方法信息
type BindingMethodInfo struct {
	// 方法名称
	MethodName string

	// 接收者类型
	ReceiverType string

	// 绑定源
	Source string

	// 绑定模型
	Model string

	// 是否有验证
	HasValidation bool

	// 验证组
	ValidationGroups []string

	// 验证规则
	ValidationRules string

	// 是否禁止未知字段
	DisallowUnknownFields bool

	// 是否中止验证失败的请求
	AbortOnFailure bool

	// 位置信息
	Position anntypes.Position
}

// extractBindingInfo 提取绑定信息
func (p *BindingProcessor) extractBindingInfo(anno anntypes.Annotation) BindingMethodInfo {
	info := BindingMethodInfo{
		MethodName:            anno.GetTargetName(),
		Position:              anno.GetPosition(),
		Source:                anntypes.GetStringAttribute(anno.GetAttributes(), "source", "json"),
		Model:                 anntypes.GetStringAttribute(anno.GetAttributes(), "model", ""),
		AbortOnFailure:        anntypes.GetBoolAttribute(anno.GetAttributes(), "abortOnFailure", true),
		DisallowUnknownFields: anntypes.GetBoolAttribute(anno.GetAttributes(), "disallowUnknownFields", false),
	}

	// 如果是注释注解，获取接收者类型
	if commentAnno, ok := anno.(*anntypes.CommentAnnotation); ok {
		info.ReceiverType = commentAnno.ReceiverType
	}

	return info
}

// extractBindingTagValue 从结构体标签中提取指定键的值
func extractBindingTagValue(tag, key string) string {
	tagContent := reflect.StructTag(tag)
	if val, ok := tagContent.Lookup(key); ok {
		return strings.Split(val, ",")[0]
	}
	return ""
}

// generateBindingCode 生成绑定代码
func (p *BindingProcessor) generateBindingCode(pkgPath, controller string, methods []BindingMethodInfo) error {
	// 准备模板数据
	data := bindingTemplateData{
		PackageName: BindingPackageGenerator.GetPackageName(pkgPath),
		Imports: []string{
			"github.com/gin-gonic/gin",
			"github.com/isBlue-5/grain/pkg/web/binding",
			"github.com/isBlue-5/grain/pkg/web/validate",
			"github.com/isBlue-5/grain/pkg/web/render",
		},
		ControllerName: controller,
		Methods:        methods,
	}

	// 解析模板
	tmpl, err := template.New("binding").Parse(bindingTemplate)
	if err != nil {
		return fmt.Errorf("解析模板失败: %w", err)
	}

	// 生成代码
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("执行模板失败: %w", err)
	}

	// 创建输出目录 - 使用统一的路径解析
	outDir := p.resolveOutputPath(pkgPath, "binding")

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 写入文件
	outFile := filepath.Join(outDir, fmt.Sprintf("%s_service_gen.go", strings.ToLower(controller)))
	if err := os.WriteFile(outFile, []byte(buf.String()), 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}

// bindingTemplateData 绑定模板数据
type bindingTemplateData struct {
	PackageName    string
	Imports        []string
	ControllerName string
	Methods        []BindingMethodInfo
}

// 绑定代码模板
const bindingTemplate = `// Code generated by gRain. DO NOT EDIT.
package {{.PackageName}}

import (
	{{range .Imports}}
	"{{.}}"
	{{end}}
)

// 绑定包装器，为控制器方法添加请求绑定和验证功能
type {{.ControllerName}}BindingWrapper struct {
	controller *{{.ControllerName}}
}

// NewBindingWrapper 创建绑定包装器
func New{{.ControllerName}}BindingWrapper(controller *{{.ControllerName}}) *{{.ControllerName}}BindingWrapper {
	return &{{.ControllerName}}BindingWrapper{
		controller: controller,
	}
}

{{range .Methods}}
// {{.MethodName}} 绑定包装器
func (w *{{$.ControllerName}}BindingWrapper) {{.MethodName}}(c *gin.Context) {
	{{if .Model}}
	// 创建模型实例
	var model {{.Model}}
	
	// 绑定请求
	bindOpts := []binding.BindingOption{}
	{{if .DisallowUnknownFields}}
	bindOpts = append(bindOpts, binding.DisallowUnknownFields())
	{{end}}
	
	if err := binding.Bind(c, &model, "{{.Source}}", bindOpts...); err != nil {
		render.Error(c, 400, err)
		{{if .AbortOnFailure}}
		c.Abort()
		{{end}}
		return
	}
	
	{{if .HasValidation}}
	// 验证请求
	validator := validate.New()
	if errs := validator.Validate(&model); len(errs) > 0 {
		render.Error(c, 400, errs)
		{{if .AbortOnFailure}}
		c.Abort()
		{{end}}
		return
	}
	{{end}}
	
	// 调用原始方法
	w.controller.{{.MethodName}}(c, &model)
	{{else}}
	// 直接调用原始方法，没有绑定模型
	w.controller.{{.MethodName}}(c)
	{{end}}
}
{{end}}
`

// resolveOutputPath 统一的输出路径解析工具，避免深层嵌套和绝对路径问题
func (p *BindingProcessor) resolveOutputPath(pkgPath, subDir string) string {
	// 如果是测试环境或包含绝对路径，使用简化的输出目录
	if strings.Contains(pkgPath, os.TempDir()) || filepath.IsAbs(pkgPath) {
		return filepath.Join(p.outputDir, subDir)
	}

	// 生产环境：使用相对路径
	cleanPkgPath := strings.TrimPrefix(pkgPath, "./")
	cleanPkgPath = strings.TrimPrefix(cleanPkgPath, "../")

	// 避免太深的嵌套，只使用最后一级目录
	if strings.Contains(cleanPkgPath, "/") {
		cleanPkgPath = filepath.Base(cleanPkgPath)
	}

	return filepath.Join(p.outputDir, cleanPkgPath, subDir)
}

// resolvePackagePath 解析包路径，返回相对路径而不是绝对路径
func (p *BindingProcessor) resolvePackagePath(filename string) string {
	absPath := filepath.Dir(filename)

	// 尝试获取相对路径
	workDir, err := os.Getwd()
	if err == nil {
		if relPath, err := filepath.Rel(workDir, absPath); err == nil && !strings.HasPrefix(relPath, "..") {
			return relPath
		}
	}

	// 如果无法获取相对路径，使用目录名
	return filepath.Base(absPath)
}
