package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/isBlue-5/grain/pkg/annotation/processor"
)

func main() {
	var (
		baseline = flag.Bool("baseline", false, "运行性能基线测试")
		compare  = flag.Bool("compare", false, "对比传统解析器和新解析器性能")
		safety   = flag.Bool("safety", false, "运行类型安全性测试")
		output   = flag.String("output", "", "保存结果到文件")
	)

	flag.Parse()

	fmt.Println("=== gRain 框架性能测试工具 ===")
	fmt.Println("版本: v1.0")
	fmt.Println("测试日期:", "2024年12月19日")
	fmt.Println()

	if *baseline {
		runBaselineTest(*output)
	}

	if *compare {
		runComparisonTest()
	}

	if *safety {
		runTypeSafetyTest()
	}

	if !*baseline && !*compare && !*safety {
		printUsage()
	}
}

func runBaselineTest(outputFile string) {
	fmt.Println("🔍 运行性能基线测试...")

	// 测量当前解析器性能
	baseline := processor.MeasureCurrentPerformance()
	baseline.PrintBaseline()

	if outputFile != "" {
		err := baseline.SaveToFile(outputFile)
		if err != nil {
			fmt.Printf("保存基线失败: %v\n", err)
		}
	}
}

func runComparisonTest() {
	fmt.Println("⚖️  运行性能对比测试...")

	// 测量传统解析器性能
	fmt.Println("测量传统解析器性能...")
	legacyBaseline := processor.MeasureCurrentPerformance()

	// 测量新解析器性能（模拟）
	fmt.Println("测量新解析器性能...")
	newBaseline := processor.MeasureNewParserPerformance()

	// 打印对比结果
	legacyBaseline.CompareWith(newBaseline)

	// 计算总体改进
	calculateOverallImprovement(legacyBaseline, newBaseline)
}

func runTypeSafetyTest() {
	fmt.Println("🛡️  运行类型安全性测试...")

	// 测量传统解析器的类型安全性
	legacyMetrics := processor.MeasureTypeSafety(false)
	legacyMetrics.PrintTypeSafetyReport("传统解析器")

	// 测量新解析器的类型安全性
	newMetrics := processor.MeasureTypeSafety(true)
	newMetrics.PrintTypeSafetyReport("新一代解析器")

	// 对比类型安全性改进
	compareTypeSafety(legacyMetrics, newMetrics)
}

func calculateOverallImprovement(legacy, new *processor.PerformanceBaseline) {
	fmt.Println("=== 总体改进评估 ===")

	// 计算总体评分
	legacyScore := calculatePerformanceScore(legacy)
	newScore := calculatePerformanceScore(new)

	improvement := (newScore - legacyScore) / legacyScore * 100

	fmt.Printf("传统解析器评分: %.1f\n", legacyScore)
	fmt.Printf("新解析器评分: %.1f\n", newScore)
	fmt.Printf("总体改进: %.1f%%\n", improvement)

	if improvement > 50 {
		fmt.Println("🎉 优化效果显著！")
	} else if improvement > 20 {
		fmt.Println("✅ 优化效果良好")
	} else {
		fmt.Println("⚠️  优化效果一般")
	}

	fmt.Println("=====================")
}

func calculatePerformanceScore(baseline *processor.PerformanceBaseline) float64 {
	// 综合评分算法：
	// 解析速度权重40%，内存使用权重20%，反射操作权重40%

	speedScore := baseline.ParseSpeed * 0.4
	memoryScore := (1.0 / (float64(baseline.MemoryUsage) / 1024 / 1024)) * 20 * 0.2 // 内存越少分数越高
	reflectionScore := (1.0 / float64(baseline.ReflectionOps)) * 1000 * 0.4         // 反射越少分数越高

	return speedScore + memoryScore + reflectionScore
}

func compareTypeSafety(legacy, new *processor.TypeSafetyMetrics) {
	fmt.Println("=== 类型安全性改进对比 ===")

	improvements := []string{}

	if !legacy.TypeAliasSupport && new.TypeAliasSupport {
		improvements = append(improvements, "✅ 新增类型别名支持")
	}

	if !legacy.EmbeddedTypeSupport && new.EmbeddedTypeSupport {
		improvements = append(improvements, "✅ 新增嵌入类型支持")
	}

	if !legacy.CrossPackageSupport && new.CrossPackageSupport {
		improvements = append(improvements, "✅ 新增跨包类型支持")
	}

	if !legacy.InterfaceCheck && new.InterfaceCheck {
		improvements = append(improvements, "✅ 新增接口实现检查")
	}

	accuracyImprovement := new.TypeAccuracy - legacy.TypeAccuracy
	if accuracyImprovement > 0 {
		improvements = append(improvements, fmt.Sprintf("✅ 类型准确性提升 %.1f%%", accuracyImprovement))
	}

	if len(improvements) == 0 {
		fmt.Println("无类型安全性改进")
	} else {
		fmt.Println("类型安全性改进:")
		for _, improvement := range improvements {
			fmt.Printf("  %s\n", improvement)
		}
	}

	fmt.Println("===============================")
}

func printUsage() {
	fmt.Println("使用方法:")
	fmt.Println("  -baseline    运行性能基线测试")
	fmt.Println("  -compare     对比传统解析器和新解析器性能")
	fmt.Println("  -safety      运行类型安全性测试")
	fmt.Println("  -output      保存结果到文件")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  go run cmd/performance-test/main.go -baseline -output baseline.json")
	fmt.Println("  go run cmd/performance-test/main.go -compare")
	fmt.Println("  go run cmd/performance-test/main.go -safety")

	os.Exit(1)
}
