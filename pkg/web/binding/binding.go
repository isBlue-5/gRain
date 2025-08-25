// Package binding 提供请求绑定和参数验证功能
package binding

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// 常量定义
const (
	MIMEJSON               = "application/json"
	MIMEXML                = "application/xml"
	MIMEForm               = "application/x-www-form-urlencoded"
	MIMEMultipartForm      = "multipart/form-data"
	MIMEPlain              = "text/plain"
	MIMEPOSTForm           = "application/x-www-form-urlencoded"
	defaultMemory          = 32 << 20 // 32 MB
	defaultMaxMultipartMem = 8 << 20  // 8 MB
)

var (
	// ErrEmptyBody 请求体为空错误
	ErrEmptyBody = errors.New("request body is empty")

	// ErrUnsupportedMediaType 不支持的媒体类型错误
	ErrUnsupportedMediaType = errors.New("unsupported media type")

	// ErrInvalidBinding 无效绑定错误
	ErrInvalidBinding = errors.New("invalid binding target")

	// ErrMissingField 缺少字段错误
	ErrMissingField = errors.New("required field missing")
)

// Binding 请求绑定接口
type Binding interface {
	Name() string
	Bind(*http.Request, interface{}) error
}

// BindingOptions 绑定选项
type BindingOptions struct {
	// 最大内存大小（用于表单和文件上传）
	MaxMemory int64

	// 是否允许未知字段（JSON解析）
	DisallowUnknownFields bool

	// 必需字段列表（字段名）
	RequiredFields []string
}

// BindingOption 绑定选项函数
type BindingOption func(*BindingOptions)

// WithMaxMemory 设置最大内存大小
func WithMaxMemory(size int64) BindingOption {
	return func(o *BindingOptions) {
		o.MaxMemory = size
	}
}

// DisallowUnknownFields 禁止未知字段
func DisallowUnknownFields() BindingOption {
	return func(o *BindingOptions) {
		o.DisallowUnknownFields = true
	}
}

// RequireFields 设置必需字段
func RequireFields(fields ...string) BindingOption {
	return func(o *BindingOptions) {
		o.RequiredFields = fields
	}
}

// defaultOptions 默认绑定选项
func defaultOptions() *BindingOptions {
	return &BindingOptions{
		MaxMemory:             defaultMaxMultipartMem,
		DisallowUnknownFields: false,
		RequiredFields:        nil,
	}
}

// JSONBinding JSON绑定
type JSONBinding struct {
	DisallowUnknownFields bool
	RequiredFields        []string
}

// Name 绑定名称
func (b *JSONBinding) Name() string {
	return "json"
}

// Bind 绑定请求
func (b *JSONBinding) Bind(req *http.Request, obj interface{}) error {
	if req == nil || req.Body == nil {
		return ErrEmptyBody
	}

	// 检查内容类型
	contentType := req.Header.Get("Content-Type")
	if contentType != "" {
		contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	}

	if contentType != MIMEJSON {
		return ErrUnsupportedMediaType
	}

	// JSON解码
	decoder := json.NewDecoder(req.Body)
	if b.DisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}

	// 解码到目标对象
	if err := decoder.Decode(obj); err != nil {
		return err
	}

	// 检查必需字段
	if len(b.RequiredFields) > 0 {
		if err := checkRequiredFields(obj, b.RequiredFields); err != nil {
			return err
		}
	}

	return nil
}

// XMLBinding XML绑定
type XMLBinding struct {
	RequiredFields []string
}

// Name 绑定名称
func (b *XMLBinding) Name() string {
	return "xml"
}

// Bind 绑定请求
func (b *XMLBinding) Bind(req *http.Request, obj interface{}) error {
	if req == nil || req.Body == nil {
		return ErrEmptyBody
	}

	// 检查内容类型
	contentType := req.Header.Get("Content-Type")
	if contentType != "" {
		contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	}

	if contentType != MIMEXML {
		return ErrUnsupportedMediaType
	}

	// XML解码
	decoder := xml.NewDecoder(req.Body)
	if err := decoder.Decode(obj); err != nil {
		return err
	}

	// 检查必需字段
	if len(b.RequiredFields) > 0 {
		if err := checkRequiredFields(obj, b.RequiredFields); err != nil {
			return err
		}
	}

	return nil
}

// FormBinding 表单绑定
type FormBinding struct {
	MaxMemory      int64
	RequiredFields []string
}

// Name 绑定名称
func (b *FormBinding) Name() string {
	return "form"
}

// Bind 绑定请求
func (b *FormBinding) Bind(req *http.Request, obj interface{}) error {
	if req == nil {
		return ErrEmptyBody
	}

	// 检查内容类型
	contentType := req.Header.Get("Content-Type")
	if contentType != "" {
		contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	}

	if contentType != MIMEForm && contentType != MIMEPOSTForm {
		return ErrUnsupportedMediaType
	}

	// 解析表单
	if err := req.ParseForm(); err != nil {
		return err
	}

	// 将表单值映射到结构体
	if err := mapForm(obj, req.Form); err != nil {
		return err
	}

	// 检查必需字段
	if len(b.RequiredFields) > 0 {
		if err := checkRequiredFields(obj, b.RequiredFields); err != nil {
			return err
		}
	}

	return nil
}

// MultipartFormBinding 多部分表单绑定
type MultipartFormBinding struct {
	MaxMemory      int64
	RequiredFields []string
}

// Name 绑定名称
func (b *MultipartFormBinding) Name() string {
	return "multipart"
}

// Bind 绑定请求
func (b *MultipartFormBinding) Bind(req *http.Request, obj interface{}) error {
	if req == nil {
		return ErrEmptyBody
	}

	// 检查内容类型
	contentType := req.Header.Get("Content-Type")
	if contentType != "" {
		contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	}

	if !strings.HasPrefix(contentType, MIMEMultipartForm) {
		return ErrUnsupportedMediaType
	}

	// 设置最大内存
	maxMemory := defaultMemory
	if b.MaxMemory > 0 {
		maxMemory = int(b.MaxMemory)
	}

	// 解析多部分表单
	if err := req.ParseMultipartForm(int64(maxMemory)); err != nil {
		return err
	}

	// 将表单值映射到结构体
	if err := mapForm(obj, req.MultipartForm.Value); err != nil {
		return err
	}

	// 处理上传文件
	if err := mapFiles(obj, req.MultipartForm.File); err != nil {
		return err
	}

	// 检查必需字段
	if len(b.RequiredFields) > 0 {
		if err := checkRequiredFields(obj, b.RequiredFields); err != nil {
			return err
		}
	}

	return nil
}

// QueryBinding 查询参数绑定
type QueryBinding struct {
	RequiredFields []string
}

// Name 绑定名称
func (b *QueryBinding) Name() string {
	return "query"
}

// Bind 绑定请求
func (b *QueryBinding) Bind(req *http.Request, obj interface{}) error {
	if req == nil {
		return ErrEmptyBody
	}

	// 解析查询参数
	values := req.URL.Query()

	// 将查询参数映射到结构体
	if err := mapForm(obj, values); err != nil {
		return err
	}

	// 检查必需字段
	if len(b.RequiredFields) > 0 {
		if err := checkRequiredFields(obj, b.RequiredFields); err != nil {
			return err
		}
	}

	return nil
}

// URIBinding URI参数绑定
type URIBinding struct {
	RequiredFields []string
}

// Name 绑定名称
func (b *URIBinding) Name() string {
	return "uri"
}

// Bind 绑定请求
func (b *URIBinding) Bind(req *http.Request, obj interface{}) error {
	// 从请求上下文中获取gin.Context
	c, ok := req.Context().Value("ginContext").(*gin.Context)
	if !ok {
		return errors.New("无法从请求中获取gin上下文")
	}

	// 提取URI参数
	params := make(map[string][]string)
	for _, p := range c.Params {
		params[p.Key] = []string{p.Value}
	}

	// 将参数映射到结构体
	if err := mapForm(obj, params); err != nil {
		return err
	}

	// 检查必需字段
	return checkRequiredFields(obj, b.RequiredFields)
}

// CreateBinding 创建绑定器
func CreateBinding(bindType string, options ...BindingOption) Binding {
	// 应用选项
	opts := defaultOptions()
	for _, option := range options {
		option(opts)
	}

	// 根据类型创建绑定器
	switch bindType {
	case "json":
		return &JSONBinding{
			DisallowUnknownFields: opts.DisallowUnknownFields,
			RequiredFields:        opts.RequiredFields,
		}
	case "xml":
		return &XMLBinding{
			RequiredFields: opts.RequiredFields,
		}
	case "form":
		return &FormBinding{
			MaxMemory:      opts.MaxMemory,
			RequiredFields: opts.RequiredFields,
		}
	case "multipart":
		return &MultipartFormBinding{
			MaxMemory:      opts.MaxMemory,
			RequiredFields: opts.RequiredFields,
		}
	case "query":
		return &QueryBinding{
			RequiredFields: opts.RequiredFields,
		}
	case "uri":
		return &URIBinding{
			RequiredFields: opts.RequiredFields,
		}
	default:
		return &JSONBinding{
			DisallowUnknownFields: opts.DisallowUnknownFields,
			RequiredFields:        opts.RequiredFields,
		}
	}
}

// MustBind 绑定请求并处理错误（如出错则中止请求）
func MustBind(c *gin.Context, obj interface{}, bindType string, options ...BindingOption) bool {
	// 将gin.Context存入http.Request的上下文中，以便在Bind方法中获取
	reqWithCtx := c.Request.WithContext(context.WithValue(c.Request.Context(), "ginContext", c))
	c.Request = reqWithCtx

	binding := CreateBinding(bindType, options...)
	if err := binding.Bind(c.Request, obj); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		c.Abort()
		return false
	}
	return true
}

// Bind 绑定请求（不中止请求）
func Bind(c *gin.Context, obj interface{}, bindType string, options ...BindingOption) error {
	// 将gin.Context存入http.Request的上下文中，以便在Bind方法中获取
	reqWithCtx := c.Request.WithContext(context.WithValue(c.Request.Context(), "ginContext", c))
	c.Request = reqWithCtx

	binding := CreateBinding(bindType, options...)
	return binding.Bind(c.Request, obj)
}

// mapForm 将表单值映射到结构体
func mapForm(ptr interface{}, form map[string][]string) error {
	if ptr == nil {
		return ErrInvalidBinding
	}

	// 获取目标对象的反射值
	value := reflect.ValueOf(ptr)
	if value.Kind() != reflect.Ptr {
		return errors.New("binding target must be a pointer")
	}

	// 解引用指针
	if value.IsNil() {
		return errors.New("nil pointer passed to mapForm")
	}

	// 获取目标结构体
	value = value.Elem()
	if value.Kind() != reflect.Struct {
		return errors.New("binding target must be a struct pointer")
	}

	// 获取类型信息
	t := value.Type()

	// 遍历字段
	for i := 0; i < value.NumField(); i++ {
		field := t.Field(i)

		// 跳过非导出字段
		if field.PkgPath != "" {
			continue
		}

		// 获取表单字段名（优先使用form标签，其次是json标签，最后是字段名）
		fieldName := field.Name
		formName := fieldName

		// 查找form标签
		if formTag := field.Tag.Get("form"); formTag != "" {
			formName = formTag
		} else if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			// 解析JSON标签，取第一个逗号前的部分
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				formName = parts[0]
			}
		}

		// 如果标签是"-"，跳过该字段
		if formName == "-" {
			continue
		}

		// 查找表单值
		inputValue, exists := form[formName]
		if !exists {
			// 尝试查找小写开头的字段名
			lcFieldName := strings.ToLower(string(fieldName[0])) + fieldName[1:]
			inputValue, exists = form[lcFieldName]

			if !exists {
				continue
			}
		}

		// 设置字段值
		fieldValue := value.Field(i)
		if !fieldValue.CanSet() {
			continue
		}

		// 处理不同类型
		setFieldValue(fieldValue, inputValue)
	}

	return nil
}

// mapFiles 将上传的文件映射到结构体
func mapFiles(ptr interface{}, files map[string][]*multipart.FileHeader) error {
	if ptr == nil {
		return ErrInvalidBinding
	}

	// 获取目标对象的反射值
	value := reflect.ValueOf(ptr)
	if value.Kind() != reflect.Ptr {
		return errors.New("binding target must be a pointer")
	}

	// 解引用指针
	if value.IsNil() {
		return errors.New("nil pointer passed to mapFiles")
	}

	// 获取目标结构体
	value = value.Elem()
	if value.Kind() != reflect.Struct {
		return errors.New("binding target must be a struct pointer")
	}

	// 获取类型信息
	t := value.Type()

	// 遍历字段
	for i := 0; i < value.NumField(); i++ {
		field := t.Field(i)

		// 跳过非导出字段
		if field.PkgPath != "" {
			continue
		}

		// 获取表单字段名（优先使用form标签，其次是json标签，最后是字段名）
		fieldName := field.Name
		formName := fieldName

		// 查找form标签
		if formTag := field.Tag.Get("form"); formTag != "" {
			formName = formTag
		} else if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			// 解析JSON标签，取第一个逗号前的部分
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				formName = parts[0]
			}
		}

		// 如果标签是"-"，跳过该字段
		if formName == "-" {
			continue
		}

		// 查找文件
		fileHeaders, exists := files[formName]
		if !exists {
			// 尝试查找小写开头的字段名
			lcFieldName := strings.ToLower(string(fieldName[0])) + fieldName[1:]
			fileHeaders, exists = files[lcFieldName]

			if !exists {
				continue
			}
		}

		// 设置字段值
		fieldValue := value.Field(i)
		if !fieldValue.CanSet() {
			continue
		}

		// 处理文件字段
		fieldType := fieldValue.Type()

		// 处理单个文件
		if fieldType == reflect.TypeOf((*multipart.FileHeader)(nil)) {
			if len(fileHeaders) > 0 {
				fieldValue.Set(reflect.ValueOf(fileHeaders[0]))
			}
			continue
		}

		// 处理文件切片
		if fieldType == reflect.TypeOf([]*multipart.FileHeader{}) {
			fieldValue.Set(reflect.ValueOf(fileHeaders))
			continue
		}
	}

	return nil
}

// setFieldValue 设置字段值
func setFieldValue(field reflect.Value, values []string) {
	if len(values) == 0 {
		return
	}

	// 获取第一个值
	value := values[0]

	// 根据字段类型设置值
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)

	case reflect.Bool:
		if value == "true" || value == "1" || value == "yes" || value == "on" {
			field.SetBool(true)
		} else {
			field.SetBool(false)
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			field.SetInt(intVal)
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if uintVal, err := strconv.ParseUint(value, 10, 64); err == nil {
			field.SetUint(uintVal)
		}

	case reflect.Float32, reflect.Float64:
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			field.SetFloat(floatVal)
		}

	case reflect.Slice:
		// 处理切片类型
		elemType := field.Type().Elem()

		// 创建切片
		sliceValue := reflect.MakeSlice(field.Type(), len(values), len(values))

		// 设置每个元素
		for i, val := range values {
			elem := sliceValue.Index(i)

			// 根据元素类型设置值
			switch elemType.Kind() {
			case reflect.String:
				elem.SetString(val)

			case reflect.Bool:
				if val == "true" || val == "1" || val == "yes" || val == "on" {
					elem.SetBool(true)
				} else {
					elem.SetBool(false)
				}

			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				if intVal, err := strconv.ParseInt(val, 10, 64); err == nil {
					elem.SetInt(intVal)
				}

			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				if uintVal, err := strconv.ParseUint(val, 10, 64); err == nil {
					elem.SetUint(uintVal)
				}

			case reflect.Float32, reflect.Float64:
				if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
					elem.SetFloat(floatVal)
				}
			}
		}

		// 设置切片
		field.Set(sliceValue)
	}
}

// checkRequiredFields 检查必需字段
func checkRequiredFields(obj interface{}, requiredFields []string) error {
	if len(requiredFields) == 0 {
		return nil
	}
	value := reflect.ValueOf(obj)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	if value.Kind() != reflect.Struct {
		return errors.New("binding target must be a struct or struct pointer")
	}

	// 获取类型信息
	t := value.Type()

	// 创建字段映射
	fieldMap := make(map[string]reflect.Value)

	// 遍历字段
	for i := 0; i < value.NumField(); i++ {
		field := t.Field(i)

		// 跳过非导出字段
		if field.PkgPath != "" {
			continue
		}

		// 获取字段名（优先使用form标签，其次是json标签，最后是字段名）
		fieldName := field.Name
		formName := fieldName

		// 查找form标签
		if formTag := field.Tag.Get("form"); formTag != "" {
			formName = formTag
		} else if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			// 解析JSON标签，取第一个逗号前的部分
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				formName = parts[0]
			}
		}

		// 将字段添加到映射
		fieldMap[formName] = value.Field(i)
	}

	// 检查必需字段
	for _, name := range requiredFields {
		field, exists := fieldMap[name]
		if !exists {
			return fmt.Errorf("%w: %s", ErrMissingField, name)
		}

		// 检查字段是否为零值
		if isZeroValue(field) {
			return fmt.Errorf("%w: %s", ErrMissingField, name)
		}
	}

	return nil
}

// isZeroValue 检查字段是否为零值
func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan:
		return v.IsNil()
	case reflect.Struct:
		return false // 结构体需要深度检查，这里简化处理
	default:
		return false
	}
}
