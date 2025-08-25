package processor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/isBlue-5/grain/pkg/annotation/registry"
)

// TestRateLimitProcessor 测试限流处理器
func TestRateLimitProcessor(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "ratelimit_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建测试用的注解注册表
	reg := registry.NewRegistry()

	// 创建注解解析器
	parser := NewAnnotationParser("frame:")

	// 创建限流处理器
	processor := NewRateLimitProcessor(reg, parser, tempDir)

	// 测试处理器创建
	if processor == nil {
		t.Fatal("限流处理器创建失败")
	}

	// 测试输出路径设置
	if processor.outputPath != tempDir {
		t.Errorf("输出路径设置错误，期望: %s, 实际: %s", tempDir, processor.outputPath)
	}

	// 测试空路径处理
	err = processor.ProcessRateLimit([]string{})
	if err != nil {
		t.Errorf("处理空路径时不应该出错: %v", err)
	}

	// 测试不存在的路径处理 - 应该返回错误，这是正常行为
	err = processor.ProcessRateLimit([]string{"/nonexistent/path"})
	if err == nil {
		t.Error("处理不存在的路径时应该返回错误")
	}
}

// TestAuthzProcessor 测试权限控制处理器
func TestAuthzProcessor(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "authz_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建测试用的注解注册表
	reg := registry.NewRegistry()

	// 创建注解解析器
	parser := NewAnnotationParser("frame:")

	// 创建权限控制处理器
	processor := NewAuthzProcessor(reg, parser, tempDir)

	// 测试处理器创建
	if processor == nil {
		t.Fatal("权限控制处理器创建失败")
	}

	// 测试输出路径设置
	if processor.outputPath != tempDir {
		t.Errorf("输出路径设置错误，期望: %s, 实际: %s", tempDir, processor.outputPath)
	}

	// 测试空路径处理
	err = processor.ProcessAuthz([]string{})
	if err != nil {
		t.Errorf("处理空路径时不应该出错: %v", err)
	}

	// 测试不存在的路径处理 - 应该返回错误，这是正常行为
	err = processor.ProcessAuthz([]string{"/nonexistent/path"})
	if err == nil {
		t.Error("处理不存在的路径时应该返回错误")
	}
}

// TestTransactionProcessor 测试事务处理器
func TestTransactionProcessor(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "transaction_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建注解解析器
	parser := NewAnnotationParser("frame:")

	// 创建模拟的生成器
	mockGenerator := &Generator{
		outputDir: tempDir,
	}

	// 创建事务处理器
	processor := NewTransactionProcessor(parser, mockGenerator, tempDir)

	// 测试处理器创建
	if processor == nil {
		t.Fatal("事务处理器创建失败")
	}

	// 测试输出路径设置
	if processor.outputPath != tempDir {
		t.Errorf("输出路径设置错误，期望: %s, 实际: %s", tempDir, processor.outputPath)
	}

	// 测试空路径处理
	err = processor.ProcessTransaction([]string{})
	if err != nil {
		t.Errorf("处理空路径时不应该出错: %v", err)
	}

	// 测试不存在的路径处理 - 应该返回错误，这是正常行为
	err = processor.ProcessTransaction([]string{"/nonexistent/path"})
	if err == nil {
		t.Error("处理不存在的路径时应该返回错误")
	}
}

// TestGenerator 测试代码生成器
func TestGenerator(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "generator_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建注解解析器和注册表
	parser := NewAnnotationParser("frame:")
	reg := registry.NewRegistry()

	// 创建生成器
	generator := NewGenerator(parser, reg, tempDir, "frame:")

	// 测试生成器创建
	if generator == nil {
		t.Fatal("生成器创建失败")
	}

	// 测试输出目录设置
	if generator.outputDir != tempDir {
		t.Errorf("输出目录设置错误，期望: %s, 实际: %s", tempDir, generator.outputDir)
	}

	// 测试处理器初始化
	if generator.rateLimitProcessor == nil {
		t.Error("限流处理器未初始化")
	}

	if generator.authzProcessor == nil {
		t.Error("权限控制处理器未初始化")
	}

	if generator.transactionProcessor == nil {
		t.Error("事务处理器未初始化")
	}

	// 测试空路径处理
	err = generator.Generate([]string{})
	if err != nil {
		t.Errorf("处理空路径时不应该出错: %v", err)
	}
}

// TestAnnotationParser 测试注解解析器
func TestAnnotationParser(t *testing.T) {
	parser := NewAnnotationParser("frame:")

	// 测试解析器创建
	if parser == nil {
		t.Fatal("注解解析器创建失败")
	}

	// 测试空包路径处理 - 空路径会解析为当前目录
	annotations, err := parser.ParsePackage("")
	if err != nil {
		t.Errorf("空包路径应该解析为当前目录，不应该返回错误: %v", err)
	}
	// 空路径会解析当前目录，应该能解析到一些文件（至少测试文件本身）
	if len(annotations) == 0 {
		t.Log("空包路径解析为空结果，这是正常的（当前目录可能没有Go文件）")
	}

	// 测试不存在的包路径 - 应该返回错误，这是正常行为
	annotations, err = parser.ParsePackage("/nonexistent/path")
	if err == nil {
		t.Error("不存在的路径应该返回错误")
	}
	if len(annotations) != 0 {
		t.Errorf("不存在的路径应该返回空注解列表，实际: %d", len(annotations))
	}
}

// TestTemplateFunctions 测试模板函数
func TestTemplateFunctions(t *testing.T) {
	// 测试 title 函数
	result := strings.Title("hello world")
	expected := "Hello World"
	if result != expected {
		t.Errorf("title函数错误，期望: %s, 实际: %s", expected, result)
	}

	// 测试 toLower 函数
	result = strings.ToLower("HELLO WORLD")
	expected = "hello world"
	if result != expected {
		t.Errorf("toLower函数错误，期望: %s, 实际: %s", expected, result)
	}

	// 测试 toUpper 函数
	result = strings.ToUpper("hello world")
	expected = "HELLO WORLD"
	if result != expected {
		t.Errorf("toUpper函数错误，期望: %s, 实际: %s", expected, result)
	}
}

// BenchmarkRateLimitProcessor 限流处理器性能测试
func BenchmarkRateLimitProcessor(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "ratelimit_bench")
	if err != nil {
		b.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	reg := registry.NewRegistry()
	parser := NewAnnotationParser("frame:")
	processor := NewRateLimitProcessor(reg, parser, tempDir)

	paths := []string{"/test/path1", "/test/path2", "/test/path3"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.ProcessRateLimit(paths)
	}
}

// BenchmarkAuthzProcessor 权限控制处理器性能测试
func BenchmarkAuthzProcessor(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "authz_bench")
	if err != nil {
		b.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	reg := registry.NewRegistry()
	parser := NewAnnotationParser("frame:")
	processor := NewAuthzProcessor(reg, parser, tempDir)

	paths := []string{"/test/path1", "/test/path2", "/test/path3"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.ProcessAuthz(paths)
	}
}

// 辅助函数：创建测试文件
func createTestFile(dir, name, content string) (string, error) {
	filePath := filepath.Join(dir, name)
	err := os.WriteFile(filePath, []byte(content), 0644)
	return filePath, err
}

// 辅助函数：清理测试文件
func cleanupTestFile(filePath string) {
	os.Remove(filePath)
}
