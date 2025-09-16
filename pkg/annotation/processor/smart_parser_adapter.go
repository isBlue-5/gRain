// Package processor 智能解析器适配器
// 将SmartAnnotationParser集成到现有框架中
package processor

import (
	"fmt"

	"github.com/your-org/gRain/pkg/annotation/types"
)

// SmartParserAdapter 智能解析器适配器
type SmartParserAdapter struct {
	smartParser      *SmartAnnotationParser
	migrationHelper  *MigrationHelper
	enableMigration  bool // 是否启用迁移建议
	logMigrationInfo bool // 是否记录迁移信息
}

// SmartParserAdapterConfig 适配器配置
type SmartParserAdapterConfig struct {
	// 智能解析器配置
	ParserConfig *SmartAnnotationParserConfig `json:"parser_config"`

	// 适配器选项
	EnableMigration  bool `json:"enable_migration"`   // 启用迁移建议
	LogMigrationInfo bool `json:"log_migration_info"` // 记录迁移信息

	// 兼容性选项
	FallbackToLegacy bool `json:"fallback_to_legacy"` // 回退到遗留解析器
}

// NewSmartParserAdapter 创建智能解析器适配器
func NewSmartParserAdapter(config *SmartParserAdapterConfig) *SmartParserAdapter {
	if config == nil {
		config = &SmartParserAdapterConfig{
			EnableMigration:  true,
			LogMigrationInfo: true,
			FallbackToLegacy: true,
		}
	}

	// 创建智能解析器
	smartParser := NewSmartAnnotationParser(config.ParserConfig)

	// 创建迁移助手
	migrationHelper := NewMigrationHelper(smartParser)

	return &SmartParserAdapter{
		smartParser:      smartParser,
		migrationHelper:  migrationHelper,
		enableMigration:  config.EnableMigration,
		logMigrationInfo: config.LogMigrationInfo,
	}
}

// ParseAttributesWithMigration 解析属性并提供迁移建议
func (a *SmartParserAdapter) ParseAttributesWithMigration(attrStr string) ([]types.AnnotationAttribute, *MigrationInfo, error) {
	migrationInfo := &MigrationInfo{
		OriginalFormat: attrStr,
		Suggestions:    []string{},
	}

	// 尝试解析
	attrs, err := a.smartParser.ParseAnnotationAttributes(attrStr)
	if err != nil {
		migrationInfo.HasError = true
		migrationInfo.Error = err.Error()
		return nil, migrationInfo, err
	}

	// 如果启用迁移建议，分析格式并提供建议
	if a.enableMigration {
		result, validateErr := a.migrationHelper.ValidateAndSuggest(attrStr)
		if validateErr != nil {
			return attrs, migrationInfo, nil // 解析成功但验证失败，不影响结果
		}

		migrationInfo.DetectedFormat = result.Format
		migrationInfo.Suggestions = result.Suggestions

		// 如果是遗留格式，标记需要迁移
		if result.Format == "Legacy" {
			migrationInfo.NeedsMigration = true

			if a.logMigrationInfo {
				fmt.Printf("⚠️  检测到遗留格式注解: %s\n", attrStr)
				for _, suggestion := range result.Suggestions {
					fmt.Printf("💡 %s\n", suggestion)
				}
			}
		}
	}

	return attrs, migrationInfo, nil
}

// ReplaceInParser 替换现有解析器中的属性解析逻辑
func (a *SmartParserAdapter) ReplaceInParser(parser *DefaultAnnotationParser) {
	// 注意: 这个方法需要在实际集成时实现
	// 因为parseComplexAttributes可能不是全局可赋值的函数
	// 这里提供一个概念性实现

	// 可以考虑通过接口或者配置的方式来替换解析逻辑
	fmt.Println("智能解析器已准备好替换现有解析逻辑")
}

// GenerateMigrationReport 生成迁移报告
func (a *SmartParserAdapter) GenerateMigrationReport(annotations []types.CommentAnnotation) *MigrationReport {
	report := &MigrationReport{
		TotalAnnotations:  len(annotations),
		JSONFormatCount:   0,
		LegacyFormatCount: 0,
		ErrorCount:        0,
		Suggestions:       []MigrationSuggestion{},
	}

	for _, annotation := range annotations {
		// 重建原始属性字符串（简化实现）
		attrStr := a.buildAttributeString(annotation.Attributes)

		result, err := a.migrationHelper.ValidateAndSuggest(attrStr)
		if err != nil {
			report.ErrorCount++
			continue
		}

		switch result.Format {
		case "JSON":
			report.JSONFormatCount++
		case "Legacy":
			report.LegacyFormatCount++

			// 添加迁移建议
			if len(result.Suggestions) > 0 {
				suggestion := MigrationSuggestion{
					Location:        fmt.Sprintf("%s:%s", annotation.TargetType, annotation.TargetName),
					OriginalFormat:  attrStr,
					SuggestedFormat: result.Suggestions[0], // 使用第一个建议
					Reason:          "建议迁移到JSON格式以获得更好的类型安全和可维护性",
				}
				report.Suggestions = append(report.Suggestions, suggestion)
			}
		}
	}

	return report
}

// buildAttributeString 从属性数组重建属性字符串（用于报告生成）
func (a *SmartParserAdapter) buildAttributeString(attrs []types.AnnotationAttribute) string {
	if len(attrs) == 0 {
		return ""
	}

	var parts []string
	for _, attr := range attrs {
		if attr.Value == true {
			parts = append(parts, attr.Name)
		} else {
			parts = append(parts, fmt.Sprintf(`%s="%v"`, attr.Name, attr.Value))
		}
	}

	return fmt.Sprintf("%s", parts)
}

// MigrationInfo 迁移信息
type MigrationInfo struct {
	OriginalFormat string   `json:"original_format"`
	DetectedFormat string   `json:"detected_format"`
	NeedsMigration bool     `json:"needs_migration"`
	HasError       bool     `json:"has_error"`
	Error          string   `json:"error,omitempty"`
	Suggestions    []string `json:"suggestions"`
}

// MigrationReport 迁移报告
type MigrationReport struct {
	TotalAnnotations  int                   `json:"total_annotations"`
	JSONFormatCount   int                   `json:"json_format_count"`
	LegacyFormatCount int                   `json:"legacy_format_count"`
	ErrorCount        int                   `json:"error_count"`
	Suggestions       []MigrationSuggestion `json:"suggestions"`
}

// MigrationSuggestion 迁移建议
type MigrationSuggestion struct {
	Location        string `json:"location"`         // 注解位置
	OriginalFormat  string `json:"original_format"`  // 原始格式
	SuggestedFormat string `json:"suggested_format"` // 建议格式
	Reason          string `json:"reason"`           // 迁移原因
}

// GetMigrationStats 获取迁移统计信息
func (r *MigrationReport) GetMigrationStats() map[string]interface{} {
	migrationRate := 0.0
	if r.TotalAnnotations > 0 {
		migrationRate = float64(r.LegacyFormatCount) / float64(r.TotalAnnotations) * 100
	}

	return map[string]interface{}{
		"total_annotations":   r.TotalAnnotations,
		"json_format_count":   r.JSONFormatCount,
		"legacy_format_count": r.LegacyFormatCount,
		"error_count":         r.ErrorCount,
		"migration_rate":      migrationRate,
		"suggestions_count":   len(r.Suggestions),
		"migration_needed":    r.LegacyFormatCount > 0,
	}
}

// PrintMigrationReport 打印迁移报告
func (r *MigrationReport) PrintMigrationReport() {
	stats := r.GetMigrationStats()

	fmt.Println("📊 gRain注解格式迁移报告")
	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("📈 总注解数量: %d\n", stats["total_annotations"])
	fmt.Printf("✅ JSON格式: %d\n", stats["json_format_count"])
	fmt.Printf("⚠️  遗留格式: %d\n", stats["legacy_format_count"])
	fmt.Printf("❌ 解析错误: %d\n", stats["error_count"])
	fmt.Printf("📊 迁移率: %.1f%%\n", stats["migration_rate"])

	if len(r.Suggestions) > 0 {
		fmt.Println("\n💡 迁移建议:")
		for i, suggestion := range r.Suggestions {
			if i >= 10 { // 限制显示前10个建议
				fmt.Printf("... 还有 %d 个建议\n", len(r.Suggestions)-i)
				break
			}
			fmt.Printf("  %d. %s\n", i+1, suggestion.Location)
			fmt.Printf("     原始: %s\n", suggestion.OriginalFormat)
			fmt.Printf("     建议: %s\n", suggestion.SuggestedFormat)
			fmt.Printf("     原因: %s\n\n", suggestion.Reason)
		}
	}
}

// CreateMigrationPlan 创建迁移计划
func (a *SmartParserAdapter) CreateMigrationPlan(report *MigrationReport) *MigrationPlan {
	plan := &MigrationPlan{
		Priority: "medium",
		Steps:    []MigrationStep{},
	}

	// 根据迁移率确定优先级
	stats := report.GetMigrationStats()
	migrationRate := stats["migration_rate"].(float64)

	if migrationRate > 80 {
		plan.Priority = "high"
	} else if migrationRate > 50 {
		plan.Priority = "medium"
	} else {
		plan.Priority = "low"
	}

	// 添加迁移步骤
	if len(report.Suggestions) > 0 {
		plan.Steps = append(plan.Steps, MigrationStep{
			Phase:       1,
			Description: "更新注解格式为JSON标准格式",
			Actions:     []string{},
		})

		for _, suggestion := range report.Suggestions {
			plan.Steps[0].Actions = append(plan.Steps[0].Actions,
				fmt.Sprintf("更新 %s: %s -> %s",
					suggestion.Location,
					suggestion.OriginalFormat,
					suggestion.SuggestedFormat))
		}
	}

	// 添加验证步骤
	plan.Steps = append(plan.Steps, MigrationStep{
		Phase:       2,
		Description: "验证迁移结果",
		Actions: []string{
			"运行所有测试确保功能正常",
			"验证代码生成结果",
			"检查性能影响",
		},
	})

	return plan
}

// MigrationPlan 迁移计划
type MigrationPlan struct {
	Priority string          `json:"priority"` // high, medium, low
	Steps    []MigrationStep `json:"steps"`
}

// MigrationStep 迁移步骤
type MigrationStep struct {
	Phase       int      `json:"phase"`
	Description string   `json:"description"`
	Actions     []string `json:"actions"`
}
