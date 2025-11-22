package arrays

// Difference 返回在 a 中存在但在 b 中不存在的元素（差集 a - b）
// 参数:
//   - a: 第一个数组
//   - b: 第二个数组
//
// 返回:
//   - 差集结果，即在 a 中但不在 b 中的元素
func Difference[T comparable](a, b []T) []T {
	// 创建一个 map 来存储 b 中的元素，以便快速查找
	bMap := make(map[T]bool, len(b))
	for _, item := range b {
		bMap[item] = true
	}

	// 存储差集结果
	result := make([]T, 0)
	for _, item := range a {
		if !bMap[item] {
			result = append(result, item)
		}
	}

	return result
}

// SymmetricDifference 返回两个数组的对称差集（在 a 或 b 中但不同时在两者中的元素）
// 参数:
//   - a: 第一个数组
//   - b: 第二个数组
//
// 返回:
//   - 对称差集结果
func SymmetricDifference[T comparable](a, b []T) []T {
	// 创建 map 来存储两个数组的元素
	aMap := make(map[T]bool, len(a))
	bMap := make(map[T]bool, len(b))

	for _, item := range a {
		aMap[item] = true
	}
	for _, item := range b {
		bMap[item] = true
	}

	// 存储对称差集结果
	result := make([]T, 0)

	// 添加在 a 中但不在 b 中的元素
	for _, item := range a {
		if !bMap[item] {
			result = append(result, item)
		}
	}

	// 添加在 b 中但不在 a 中的元素
	for _, item := range b {
		if !aMap[item] {
			result = append(result, item)
		}
	}

	return result
}
