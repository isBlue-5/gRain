package processor

import (
	"testing"
)

// BenchmarkNextGenAdapter_Creation 基准测试NextGen适配器创建
func BenchmarkNextGenAdapter_Creation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter := NewNextGenAdapter("frame:", true)
		if adapter == nil {
			b.Fatal("适配器创建失败")
		}
	}
}

// BenchmarkNextGenAdapter_TypeAwareness 基准测试类型感知功能
func BenchmarkNextGenAdapter_TypeAwareness(b *testing.B) {
	adapter := NewNextGenAdapter("frame:", true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		enabled := adapter.IsNextGenEnabled()
		if !enabled {
			b.Fatal("NextGen应该是启用的")
		}
	}
}

// BenchmarkProcessorConfig_Creation 基准测试配置创建
func BenchmarkProcessorConfig_Creation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config := DefaultProcessorConfig()
		if config == nil {
			b.Fatal("配置创建失败")
		}
	}
}

// BenchmarkProcessorFactory_Creation 基准测试工厂创建
func BenchmarkProcessorFactory_Creation(b *testing.B) {
	config := DefaultProcessorConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		factory := NewProcessorFactory(config)
		if factory == nil {
			b.Fatal("工厂创建失败")
		}
	}
}

// BenchmarkProcessorFactory_CreateParser 基准测试解析器创建
func BenchmarkProcessorFactory_CreateParser(b *testing.B) {
	config := DefaultProcessorConfig()
	factory := NewProcessorFactory(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := factory.CreateParser("frame:")
		if parser == nil {
			b.Fatal("解析器创建失败")
		}
	}
}

// BenchmarkNextGenParser_TypeHelpers 基准测试类型辅助函数
func BenchmarkNextGenParser_TypeHelpers(b *testing.B) {
	parser := NewNextGenParser("frame:")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 测试各种类型辅助函数
		_ = parser.IsPointerType(nil)
		_ = parser.IsSliceType(nil)
		_ = parser.IsMapType(nil)
		_ = parser.IsInterfaceType(nil)
		_ = parser.IsStructType(nil)
		_ = parser.IsBasicType(nil)
	}
}

// BenchmarkAll 运行所有基准测试
func BenchmarkAll(b *testing.B) {
	b.Run("NextGenAdapter_Creation", BenchmarkNextGenAdapter_Creation)
	b.Run("NextGenAdapter_TypeAwareness", BenchmarkNextGenAdapter_TypeAwareness)
	b.Run("ProcessorConfig_Creation", BenchmarkProcessorConfig_Creation)
	b.Run("ProcessorFactory_Creation", BenchmarkProcessorFactory_Creation)
	b.Run("ProcessorFactory_CreateParser", BenchmarkProcessorFactory_CreateParser)

	b.Run("NextGenParser_TypeHelpers", BenchmarkNextGenParser_TypeHelpers)
}

// TestBenchmarkResults 测试基准测试结果
func TestBenchmarkResults(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过基准测试结果验证")
	}

	t.Log("开始基准测试结果验证...")

	// 测试NextGen适配器创建性能
	result := testing.Benchmark(BenchmarkNextGenAdapter_Creation)
	t.Logf("NextGen适配器创建性能: %s", result.String())
	if result.NsPerOp() > 10000 {
		t.Logf("警告: NextGen适配器创建较慢 (%d ns/op)", result.NsPerOp())
	}

	// 测试类型感知功能性能
	result = testing.Benchmark(BenchmarkNextGenAdapter_TypeAwareness)
	t.Logf("类型感知功能性能: %s", result.String())

	// 测试配置创建性能
	result = testing.Benchmark(BenchmarkProcessorConfig_Creation)
	t.Logf("配置创建性能: %s", result.String())

	// 测试工厂创建性能
	result = testing.Benchmark(BenchmarkProcessorFactory_Creation)
	t.Logf("工厂创建性能: %s", result.String())

	// 测试解析器创建性能
	result = testing.Benchmark(BenchmarkProcessorFactory_CreateParser)
	t.Logf("解析器创建性能: %s", result.String())

	// 测试类型辅助函数性能
	result = testing.Benchmark(BenchmarkNextGenParser_TypeHelpers)
	t.Logf("类型辅助函数性能: %s", result.String())

	t.Log("基准测试结果验证完成")
}
