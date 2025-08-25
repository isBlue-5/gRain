package logging

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// JSONFormatter JSON格式化器
type JSONFormatter struct {
	TimeFormat string
}

// Format 实现Formatter接口
func (f *JSONFormatter) Format(entry *Entry) ([]byte, error) {
	data := make(map[string]interface{}, len(entry.Fields)+4)

	// 添加基本字段
	timeFormat := f.TimeFormat
	if timeFormat == "" {
		timeFormat = time.RFC3339
	}

	data["time"] = entry.Time.Format(timeFormat)
	data["level"] = entry.Level.String()
	data["message"] = entry.Message

	if entry.TraceID != "" {
		data["trace_id"] = entry.TraceID
	}

	// 添加自定义字段
	for _, field := range entry.Fields {
		data[field.Key] = field.Value
	}

	return json.Marshal(data)
}

// TextFormatter 文本格式化器
type TextFormatter struct {
	TimeFormat string
}

// Format 实现Formatter接口
func (f *TextFormatter) Format(entry *Entry) ([]byte, error) {
	timeFormat := f.TimeFormat
	if timeFormat == "" {
		timeFormat = "2006-01-02T15:04:05.000Z07:00"
	}

	// 构建基本日志行
	parts := []string{
		entry.Time.Format(timeFormat),
		strings.ToUpper(entry.Level.String()),
		entry.Message,
	}

	// 添加字段
	if len(entry.Fields) > 0 {
		fieldParts := make([]string, 0, len(entry.Fields))
		for _, field := range entry.Fields {
			fieldParts = append(fieldParts, fmt.Sprintf("%s=%v", field.Key, field.Value))
		}
		parts = append(parts, strings.Join(fieldParts, " "))
	}

	// 添加跟踪ID
	if entry.TraceID != "" {
		parts = append(parts, fmt.Sprintf("trace_id=%s", entry.TraceID))
	}

	return []byte(strings.Join(parts, " ") + "\n"), nil
}

// SimpleFormatter 简单格式化器
type SimpleFormatter struct {
	TimeFormat string
}

// Format 实现Formatter接口
func (f *SimpleFormatter) Format(entry *Entry) ([]byte, error) {
	timeFormat := f.TimeFormat
	if timeFormat == "" {
		timeFormat = "2006-01-02 15:04:05"
	}

	// 简单格式：[时间] [级别] 消息
	format := "[%s] [%s] %s"
	args := []interface{}{
		entry.Time.Format(timeFormat),
		strings.ToUpper(entry.Level.String()),
		entry.Message,
	}

	// 如果有字段，添加到消息后面
	if len(entry.Fields) > 0 {
		fieldParts := make([]string, 0, len(entry.Fields))
		for _, field := range entry.Fields {
			fieldParts = append(fieldParts, fmt.Sprintf("%s=%v", field.Key, field.Value))
		}
		format += " %s"
		args = append(args, strings.Join(fieldParts, " "))
	}

	return []byte(fmt.Sprintf(format, args...) + "\n"), nil
}

// CustomFormatter 自定义格式化器
type CustomFormatter struct {
	Template   string
	TimeFormat string
}

// Format 实现Formatter接口
func (f *CustomFormatter) Format(entry *Entry) ([]byte, error) {
	timeFormat := f.TimeFormat
	if timeFormat == "" {
		timeFormat = time.RFC3339
	}

	// 替换模板中的占位符
	result := f.Template
	result = strings.ReplaceAll(result, "{{time}}", entry.Time.Format(timeFormat))
	result = strings.ReplaceAll(result, "{{level}}", entry.Level.String())
	result = strings.ReplaceAll(result, "{{message}}", entry.Message)
	result = strings.ReplaceAll(result, "{{trace_id}}", entry.TraceID)

	// 处理字段占位符
	for _, field := range entry.Fields {
		placeholder := fmt.Sprintf("{{%s}}", field.Key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", field.Value))
	}

	return []byte(result + "\n"), nil
}
