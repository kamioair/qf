package qf

import (
	"testing"

	"github.com/kamioair/utils/qtime"
)

// TestGetDateTime 回归测试：Context.Get[qtime.DateTime] / Get[qtime.Date]
//
// v1.4.4 缺陷：
//  1. parseDateTimeValue/parseDateValue 把解析结果写进匿名结构体字面量，
//     外层变量从未赋值，Get[qtime.DateTime] 恒返回零值；
//  2. 数字入参（JSON number → float64）经 toString 走 json.Marshal
//     产出科学计数法字符串，dateparse 解析失败，同样返回零值。
func TestGetDateTime(t *testing.T) {
	ctx, err := NewContext(map[string]any{
		// 字符串日期（dateparse 可解析格式）
		"endTime": "2099-12-31 23:59:59",
		"dateStr": "2026-11-27",
		// 数字日期（JSON 反序列化后为 float64）
		"planTime": 20991231235959,
		"dateNum":  20261127,
	}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if got := ctx.Get[qtime.DateTime]("endTime"); got != 20991231235959 {
		t.Errorf("endTime(string) = %d, want 20991231235959", got)
	}
	if got := ctx.Get[qtime.DateTime]("planTime"); got != 20991231235959 {
		t.Errorf("planTime(number/float64) = %d, want 20991231235959", got)
	}
	if got := ctx.Get[qtime.Date]("dateStr"); got != 20261127 {
		t.Errorf("dateStr(string) = %d, want 20261127", got)
	}
	if got := ctx.Get[qtime.Date]("dateNum"); got != 20261127 {
		t.Errorf("dateNum(number/float64) = %d, want 20261127", got)
	}

	// 不存在的 key 应返回零值而非 panic
	if got := ctx.Get[qtime.DateTime]("missing"); got != 0 {
		t.Errorf("missing key = %d, want 0", got)
	}
}
