package arrays

import (
	"errors"
	"reflect"
	"strconv"
	"testing"
)

func TestMap(t *testing.T) {
	t.Run("整数转字符串", func(t *testing.T) {
		source := []int{1, 2, 3, 4, 5}
		expected := []string{"1", "2", "3", "4", "5"}

		result := Map(source, func(n int) string {
			return strconv.Itoa(n)
		})

		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Map() = %v, expected %v", result, expected)
		}
	})

	t.Run("字符串转整数", func(t *testing.T) {
		source := []string{"10", "20", "30"}
		expected := []int{10, 20, 30}

		result := Map(source, func(s string) int {
			n, _ := strconv.Atoi(s)
			return n
		})

		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Map() = %v, expected %v", result, expected)
		}
	})

	t.Run("结构体字段提取", func(t *testing.T) {
		type User struct {
			ID   int
			Name string
		}

		source := []User{
			{ID: 1, Name: "Alice"},
			{ID: 2, Name: "Bob"},
			{ID: 3, Name: "Charlie"},
		}
		expected := []string{"Alice", "Bob", "Charlie"}

		result := Map(source, func(u User) string {
			return u.Name
		})

		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Map() = %v, expected %v", result, expected)
		}
	})

	t.Run("nil数组", func(t *testing.T) {
		var source []int
		result := Map(source, func(n int) string {
			return strconv.Itoa(n)
		})

		if result != nil {
			t.Errorf("Map() = %v, expected nil", result)
		}
	})

	t.Run("空数组", func(t *testing.T) {
		source := []int{}
		result := Map(source, func(n int) string {
			return strconv.Itoa(n)
		})

		if len(result) != 0 {
			t.Errorf("Map() length = %d, expected 0", len(result))
		}
	})
}

func TestMapWithError(t *testing.T) {
	t.Run("成功转换", func(t *testing.T) {
		source := []string{"1", "2", "3"}
		expected := []int{1, 2, 3}

		result, err := MapWithError(source, func(s string) (int, error) {
			return strconv.Atoi(s)
		})

		if err != nil {
			t.Errorf("MapWithError() error = %v", err)
		}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("MapWithError() = %v, expected %v", result, expected)
		}
	})

	t.Run("转换失败", func(t *testing.T) {
		source := []string{"1", "invalid", "3"}

		result, err := MapWithError(source, func(s string) (int, error) {
			return strconv.Atoi(s)
		})

		if err == nil {
			t.Errorf("MapWithError() expected error, got nil")
		}
		if result != nil {
			t.Errorf("MapWithError() = %v, expected nil on error", result)
		}
	})

	t.Run("自定义错误", func(t *testing.T) {
		source := []int{1, 2, -1, 3}

		result, err := MapWithError(source, func(n int) (int, error) {
			if n < 0 {
				return 0, errors.New("negative number not allowed")
			}
			return n * 2, nil
		})

		if err == nil {
			t.Errorf("MapWithError() expected error, got nil")
		}
		if result != nil {
			t.Errorf("MapWithError() = %v, expected nil on error", result)
		}
	})

	t.Run("nil数组", func(t *testing.T) {
		var source []string
		result, err := MapWithError(source, func(s string) (int, error) {
			return strconv.Atoi(s)
		})

		if err != nil {
			t.Errorf("MapWithError() error = %v, expected nil", err)
		}
		if result != nil {
			t.Errorf("MapWithError() = %v, expected nil", result)
		}
	})
}

func TestMapWithIndex(t *testing.T) {
	t.Run("使用索引", func(t *testing.T) {
		source := []string{"a", "b", "c"}
		expected := []string{"0:a", "1:b", "2:c"}

		result := MapWithIndex(source, func(i int, s string) string {
			return strconv.Itoa(i) + ":" + s
		})

		if !reflect.DeepEqual(result, expected) {
			t.Errorf("MapWithIndex() = %v, expected %v", result, expected)
		}
	})

	t.Run("索引计算", func(t *testing.T) {
		source := []int{10, 20, 30}
		expected := []int{10, 21, 32}

		result := MapWithIndex(source, func(i int, n int) int {
			return n + i
		})

		if !reflect.DeepEqual(result, expected) {
			t.Errorf("MapWithIndex() = %v, expected %v", result, expected)
		}
	})

	t.Run("nil数组", func(t *testing.T) {
		var source []int
		result := MapWithIndex(source, func(i int, n int) string {
			return strconv.Itoa(i) + ":" + strconv.Itoa(n)
		})

		if result != nil {
			t.Errorf("MapWithIndex() = %v, expected nil", result)
		}
	})
}
