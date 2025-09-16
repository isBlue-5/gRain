package processor

import (
	"testing"
)

// BenchmarkOriginalParser 基准测试原始解析器性能
func BenchmarkOriginalParser(b *testing.B) {
	parser := NewAnnotationParser("frame:")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 模拟解析操作
		_, err := parser.ParsePackage("../../../examples/basic_demo")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkNextGenParser 基准测试NextGen解析器性能
func BenchmarkNextGenParser(b *testing.B) {
	adapter := NewNextGenAdapter("frame:", true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 模拟解析操作
		_, err := adapter.ParsePackage("../../../examples/basic_demo")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTypeInfoRetrieval 基准测试类型信息获取
func BenchmarkTypeInfoRetrieval(b *testing.B) {
	adapter := NewNextGenAdapter("frame:", true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 测试类型信息获取
		_, err := adapter.GetEnhancedPackageInfo("../../../examples/basic_demo")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkProcessorFactory 基准测试处理器工厂性能
func BenchmarkProcessorFactory(b *testing.B) {
	config := DefaultProcessorConfig()
	factory := NewProcessorFactory(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 测试处理器创建
		processor := factory.CreateParamBindingProcessor("/tmp/test")
		if processor == nil {
			b.Fatal("处理器创建失败")
		}
	}
}

// BenchmarkProcessorUpdate 基准测试处理器更新性能
func BenchmarkProcessorUpdate(b *testing.B) {
	adapter := NewNextGenAdapter("frame:", true)
	processor := &ParamBindingProcessor{
		outputPath: "/tmp/test",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 测试处理器更新
		err := UpdateProcessorWithNextGen(processor, adapter)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkReflectionUpdate 基准测试反射更新性能
func BenchmarkReflectionUpdate(b *testing.B) {
	adapter := NewNextGenAdapter("frame:", true)

	// 创建自定义处理器用于反射测试
	type CustomProcessor struct {
		parser AnnotationParser
		name   string
	}

	processor := &CustomProcessor{
		name: "test",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 测试反射更新
		err := updateProcessorWithReflection(processor, adapter)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkComparison 对比基准测试
func BenchmarkComparison(b *testing.B) {
	b.Run("OriginalParser", BenchmarkOriginalParser)
	b.Run("NextGenParser", BenchmarkNextGenParser)
	b.Run("TypeInfoRetrieval", BenchmarkTypeInfoRetrieval)
	b.Run("ProcessorFactory", BenchmarkProcessorFactory)
	b.Run("ProcessorUpdate", BenchmarkProcessorUpdate)
	b.Run("ReflectionUpdate", BenchmarkReflectionUpdate)
}

// TestPerformanceComparison 性能对比测试
func TestPerformanceComparison(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过性能对比测试")
	}

	t.Log("开始性能对比测试...")

	// 测试原始解析器
	t.Run("OriginalParserPerformance", func(t *testing.T) {
		result := testing.Benchmark(BenchmarkOriginalParser)
		t.Logf("原始解析器性能: %s", result.String())
	})

	// 测试NextGen解析器
	t.Run("NextGenParserPerformance", func(t *testing.T) {
		result := testing.Benchmark(BenchmarkNextGenParser)
		t.Logf("NextGen解析器性能: %s", result.String())
	})

	// 测试类型信息获取
	t.Run("TypeInfoRetrievalPerformance", func(t *testing.T) {
		result := testing.Benchmark(BenchmarkTypeInfoRetrieval)
		t.Logf("类型信息获取性能: %s", result.String())
	})

	t.Log("性能对比测试完成")
}

// TestMemoryUsage 内存使用测试
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过内存使用测试")
	}

	t.Log("开始内存使用测试...")

	// 测试原始解析器内存使用
	t.Run("OriginalParserMemory", func(t *testing.T) {
		result := testing.Benchmark(func(b *testing.B) {
			b.ReportAllocs()
			BenchmarkOriginalParser(b)
		})
		t.Logf("原始解析器内存使用: %s", result.String())
	})

	// 测试NextGen解析器内存使用
	t.Run("NextGenParserMemory", func(t *testing.T) {
		result := testing.Benchmark(func(b *testing.B) {
			b.ReportAllocs()
			BenchmarkNextGenParser(b)
		})
		t.Logf("NextGen解析器内存使用: %s", result.String())
	})

	t.Log("内存使用测试完成")
}
