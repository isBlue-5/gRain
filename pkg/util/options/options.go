// Package options 提供通用的函数选项模式实现
package options

// Option 定义通用的选项函数接口
// 可用于配置各种对象的泛型模板
type Option[T any] func(*T)

// OptionWithError 带有错误返回的选项函数
// 用于需要验证选项有效性的场景
type OptionWithError[T any] func(*T) error

// Apply 将多个选项函数应用到目标对象
// 这是一个辅助函数，简化选项应用逻辑
func Apply[T any](target *T, opts ...Option[T]) {
	for _, opt := range opts {
		opt(target)
	}
}

// ApplyWithError 应用带错误返回的选项函数
// 如果任何选项函数返回错误，将立即返回该错误
func ApplyWithError[T any](target *T, opts ...OptionWithError[T]) error {
	for _, opt := range opts {
		if err := opt(target); err != nil {
			return err
		}
	}
	return nil
}

// Conditional 创建一个条件性的选项函数
// 只有当条件为真时才应用选项
func Conditional[T any](condition bool, opt Option[T]) Option[T] {
	return func(t *T) {
		if condition {
			opt(t)
		}
	}
}

// Group 将多个选项组合成单个选项
// 可用于创建预定义的选项组
func Group[T any](opts ...Option[T]) Option[T] {
	return func(t *T) {
		for _, opt := range opts {
			opt(t)
		}
	}
}
