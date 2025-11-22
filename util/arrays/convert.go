package arrays

// Map 将数组中的每个元素通过转换函数转换为另一种类型的元素
// 参数:
//   - source: 原始数组
//   - converter: 转换函数，接收原类型元素，返回目标类型元素
//
// 返回:
//   - 转换后的新数组
func Map[T any, R any](source []T, converter func(T) R) []R {
	if source == nil {
		return nil
	}

	result := make([]R, 0, len(source))
	for _, item := range source {
		result = append(result, converter(item))
	}
	return result
}

// MapWithError 将数组中的每个元素通过转换函数转换为另一种类型的元素（支持错误处理）
// 参数:
//   - source: 原始数组
//   - converter: 转换函数，接收原类型元素，返回目标类型元素和错误
//
// 返回:
//   - 转换后的新数组和可能的错误
func MapWithError[T any, R any](source []T, converter func(T) (R, error)) ([]R, error) {
	if source == nil {
		return nil, nil
	}

	result := make([]R, 0, len(source))
	for _, item := range source {
		converted, err := converter(item)
		if err != nil {
			return nil, err
		}
		result = append(result, converted)
	}
	return result, nil
}

// MapWithIndex 将数组中的每个元素通过转换函数转换为另一种类型的元素（转换函数可访问索引）
// 参数:
//   - source: 原始数组
//   - converter: 转换函数，接收索引和原类型元素，返回目标类型元素
//
// 返回:
//   - 转换后的新数组
func MapWithIndex[T any, R any](source []T, converter func(int, T) R) []R {
	if source == nil {
		return nil
	}

	result := make([]R, 0, len(source))
	for i, item := range source {
		result = append(result, converter(i, item))
	}
	return result
}
