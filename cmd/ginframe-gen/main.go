// ginframe-gen 是gRain框架的代码生成工具
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/isBlue-5/grain/pkg/annotation/processor"
	"github.com/isBlue-5/grain/pkg/annotation/registry"
)

const (
	// 注解前缀
	defaultPrefix = "frame:"

	// 版本号
	version = "0.1.0"
)

var (
	// 命令行参数
	help       bool
	showVer    bool
	verbose    bool
	prefix     string
	outputDir  string
	pkgPaths   string
	cacheDir   string
	noCache    bool
	skipErrors bool
)

// 颜色输出
var (
	errorColor     = color.New(color.FgHiRed).SprintFunc()
	warningColor   = color.New(color.FgHiYellow).SprintFunc()
	successColor   = color.New(color.FgHiGreen).SprintFunc()
	infoColor      = color.New(color.FgHiBlue).SprintFunc()
	highlightColor = color.New(color.FgHiCyan).SprintFunc()
)

func init() {
	// 定义命令行参数
	flag.BoolVar(&help, "help", false, "显示帮助信息")
	flag.BoolVar(&help, "h", false, "显示帮助信息 (简写)")
	flag.BoolVar(&showVer, "version", false, "显示版本信息")
	flag.BoolVar(&showVer, "v", false, "显示版本信息 (简写)")
	flag.BoolVar(&verbose, "verbose", false, "启用详细日志")
	flag.StringVar(&prefix, "prefix", defaultPrefix, "注解前缀")
	flag.StringVar(&outputDir, "output", ".", "输出目录")
	flag.StringVar(&pkgPaths, "pkg", "./...", "包路径，多个路径用逗号分隔")
	flag.StringVar(&cacheDir, "cache-dir", ".ginframe-cache", "缓存目录")
	flag.BoolVar(&noCache, "no-cache", false, "禁用增量生成缓存")
	flag.BoolVar(&skipErrors, "skip-errors", false, "忽略非致命错误并继续生成")
}

func main() {
	// 解析命令行参数
	flag.Parse()

	// 显示帮助
	if help {
		printHelp()
		return
	}

	// 显示版本
	if showVer {
		printVersion()
		return
	}

	// 分割包路径
	paths := splitPaths(pkgPaths)

	// 如果没有指定包路径，使用当前目录
	if len(paths) == 0 || (len(paths) == 1 && paths[0] == "./...") {
		paths = []string{"."}
	}

	// 创建注解解析器
	parser := processor.NewAnnotationParser(prefix)

	// 创建注册中心
	reg := registry.NewRegistry()

	// 创建生成器
	generator := processor.NewGenerator(parser, reg, outputDir, prefix)

	// 应用选项
	if verbose {
		processor.WithVerbose(verbose)(generator)
	}
	if cacheDir != "" {
		processor.WithCacheDir(cacheDir)(generator)
	}
	if !noCache {
		processor.WithIncremental(true)(generator)
	}

	// 生成代码
	if err := generator.Generate(paths); err != nil {
		if skipErrors {
			fmt.Fprintf(os.Stderr, "%s: %v\n%s\n",
				warningColor("警告"), err,
				warningColor("继续生成，但结果可能不完整或不正确。"))
		} else {
			fmt.Fprintf(os.Stderr, "%s: %v\n", errorColor("错误"), err)
			printErrorHelp(err)
			os.Exit(1)
		}
	} else {
		// 打印成功信息
		stats := generator.GetStats()
		fmt.Printf("%s 代码生成完成！\n", successColor("✓"))
		fmt.Printf("  - 处理文件: %s\n", highlightColor("%d", stats.TotalFiles))
		fmt.Printf("  - 生成代码: %s\n", highlightColor("已输出到 %s 目录", outputDir))

		if verbose {
			printAnnotationStats(generator)
		}
	}
}

// printErrorHelp 打印错误帮助信息
func printErrorHelp(err error) {
	errStr := err.Error()

	// 解析错误，提供针对性建议
	switch {
	case strings.Contains(errStr, "循环依赖"):
		fmt.Fprintln(os.Stderr, warningColor("\n如何修复循环依赖:"))
		fmt.Fprintln(os.Stderr, "1. 检查依赖链中可能形成环的组件")
		fmt.Fprintln(os.Stderr, "2. 考虑通过接口打破循环")
		fmt.Fprintln(os.Stderr, "3. 或者重构组件以移除循环依赖")

	case strings.Contains(errStr, "解析失败"):
		fmt.Fprintln(os.Stderr, warningColor("\n可能的语法问题:"))
		fmt.Fprintln(os.Stderr, "1. 请检查源文件是否有语法错误")
		fmt.Fprintln(os.Stderr, "2. 确保注解格式正确，如 // frame:route(...)")
		fmt.Fprintln(os.Stderr, "3. 检查结构体标签是否有语法错误")

	case strings.Contains(errStr, "不是一个包"):
		fmt.Fprintln(os.Stderr, warningColor("\n路径问题:"))
		fmt.Fprintln(os.Stderr, "1. 确保指定的路径包含有效的Go包")
		fmt.Fprintln(os.Stderr, "2. 使用 -pkg 参数指定正确的包路径")

	default:
		fmt.Fprintln(os.Stderr, warningColor("\n尝试以下操作:"))
		fmt.Fprintln(os.Stderr, "1. 使用 --verbose 参数获取更多详细信息")
		fmt.Fprintln(os.Stderr, "2. 检查您的注解语法和格式是否正确")
		fmt.Fprintln(os.Stderr, "3. 确认所有引用的包和类型都存在")
	}

	fmt.Fprintln(os.Stderr, warningColor("\n如需帮助，请运行 'ginframe-gen --help'"))
}

// printAnnotationStats 打印注解统计信息
func printAnnotationStats(generator *processor.Generator) {
	counts := generator.CountAnnotationsByType()

	if len(counts) > 0 {
		fmt.Println(infoColor("\n注解统计:"))
		for annoType, count := range counts {
			fmt.Printf("  - %s: %s\n", annoType, highlightColor("%d 个", count))
		}
	}
}

// printHelp 打印帮助信息
func printHelp() {
	fmt.Printf("%s - gRain框架代码生成工具 (v%s)\n\n",
		highlightColor("ginframe-gen"), version)

	fmt.Println(infoColor("用法:"))
	fmt.Println("  ginframe-gen [选项] [包路径]")

	fmt.Println(infoColor("\n选项:"))
	flag.PrintDefaults()

	fmt.Println(infoColor("\n示例:"))
	fmt.Println("  ginframe-gen -verbose -output ./generated ./...")
	fmt.Println("  ginframe-gen -prefix frame: -pkg ./pkg/controller,./pkg/service")

	fmt.Println(infoColor("\n注解系统:"))
	fmt.Printf("  - %s: 标记控制器类型\n", highlightColor("frame:controller"))
	fmt.Printf("  - %s: 标记HTTP路由\n", highlightColor("frame:route"))
	fmt.Printf("  - %s: 标记依赖注入\n", highlightColor("frame:inject"))
	fmt.Printf("  - %s: 标记服务\n", highlightColor("frame:service"))
	fmt.Printf("  - %s: 标记方法日志\n", highlightColor("frame:log"))
	fmt.Printf("  - %s: 标记事务控制\n", highlightColor("frame:transaction"))
	fmt.Printf("  - %s: 标记参数验证\n", highlightColor("frame:validation"))
	fmt.Printf("  - %s: 标记方法缓存\n", highlightColor("frame:cache"))
}

// printVersion 打印版本信息
func printVersion() {
	fmt.Printf("%s v%s\n", highlightColor("ginframe-gen"), version)
}

// splitPaths 分割包路径
func splitPaths(pathsStr string) []string {
	// 按逗号分割
	paths := strings.Split(pathsStr, ",")

	// 规范化路径
	for i, path := range paths {
		path = strings.TrimSpace(path)

		// 如果是相对路径且不以./开头，添加./前缀
		if !strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "./") && path != "..." && path != "./..." {
			path = "./" + path
		}

		paths[i] = path
	}

	return paths
}
