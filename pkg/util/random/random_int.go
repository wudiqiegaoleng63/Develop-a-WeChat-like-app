package random

import (
	"errors"
	"math/rand/v2"
)

// GetRandomInt 生成指定位数的随机数字字符串
// n: 位数，有效范围 1-18
func GetRandomInt(n int) (int, error) {
	// 参数校验
	if n <= 0 {
		return 0, errors.New("位数必须大于0")
	}
	if n > 18 {
		return 0, errors.New("位数不能超过18（防止int溢出）")
	}

	min := pow(10, n-1)
	max := pow(10, n) - 1
	return rand.IntN(max-min+1) + min, nil
}

func pow(base, n int) int {
	result := 1
	for i := 0; i < n; i++ {
		result *= base
	}
	return result
}