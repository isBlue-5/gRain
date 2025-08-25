package processor

import (
	"testing"

	"github.com/grain-framework/grain/pkg/annotation/registry"
)

// BenchmarkAnnotationParsing 注解解析性能基准测试
func BenchmarkAnnotationParsing(b *testing.B) {
	parser := NewAnnotationParser("@frame:")

	// 准备测试数据 - 这里只是示例，实际解析会从文件读取
	_ = `// @frame:controller
type UserController struct{}

// @frame:route(method="GET", path="/users")
func (c *UserController) ListUsers(ctx *gin.Context) {
	ctx.JSON(200, gin.H{"message": "users"})
}

// @frame:service
type UserService struct{}

// @frame:method
func (s *UserService) GetUsers() []User {
	return []User{}
}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseFile("benchmark_test.go")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRegistryOperations 注册表操作性能基准测试
func BenchmarkRegistryOperations(b *testing.B) {
	reg := registry.NewRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 模拟注册操作
		reg.GetAll()
	}
}

// BenchmarkCodeGeneration 代码生成性能基准测试
func BenchmarkCodeGeneration(b *testing.B) {
	parser := NewAnnotationParser("@frame:")
	reg := registry.NewRegistry()
	generator := NewGenerator(parser, reg, "/tmp", "@frame:")

	// 准备测试文件
	testFiles := []string{"test_controller.go", "test_service.go"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := generator.Generate(testFiles)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryUsage 内存使用基准测试
func BenchmarkMemoryUsage(b *testing.B) {
	b.ReportAllocs()

	parser := NewAnnotationParser("@frame:")
	reg := registry.NewRegistry()
	generator := NewGenerator(parser, reg, "/tmp", "@frame:")

	for i := 0; i < b.N; i++ {
		// 模拟内存分配
		_ = generator
		_ = reg
		_ = parser
	}
}

// BenchmarkConcurrentProcessing 并发处理性能基准测试
func BenchmarkConcurrentProcessing(b *testing.B) {
	parser := NewAnnotationParser("@frame:")
	_ = registry.NewRegistry()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// 模拟并发注解解析
			_, err := parser.ParseFile("concurrent_test.go")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
