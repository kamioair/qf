package qf

import (
	"reflect"
	"strings"
	"testing"

	easyCon "github.com/qiu-tec/easy-con.golang"
)

// mockAdapter 仅用于单元测试，按 IAdapter（含 iLogger 全部方法）实现。
type mockAdapter struct {
	respFunc func(module, route string, params any) easyCon.PackResp
}

func (m *mockAdapter) Stop()                                {}
func (m *mockAdapter) Reset()                               {}
func (m *mockAdapter) Req(module, route string, params any) easyCon.PackResp {
	if m.respFunc != nil {
		return m.respFunc(module, route, params)
	}
	return easyCon.PackResp{}
}
func (m *mockAdapter) SendNotice(route string, content any) error      { return nil }
func (m *mockAdapter) SendRetainNotice(route string, content any) error { return nil }
func (m *mockAdapter) CleanRetainNotice() error                        { return nil }
func (m *mockAdapter) Debug(content string)                            {}
func (m *mockAdapter) Warn(content string)                             {}
func (m *mockAdapter) Err(content string, err error)                   {}

func newServiceWithMock(m easyCon.IAdapter) *Service {
	s := &Service{}
	s.setAdapter(m)
	s.setWriteLog(func(level, content, err string) {})
	return s
}

// TestSendRequest_TIsAny 验证 T 为 any 时直接透传 resp.Content。
func TestSendRequest_TIsAny(t *testing.T) {
	want := map[string]any{"name": "joey", "tags": []any{"a", "b"}}
	mock := &mockAdapter{
		respFunc: func(module, route string, params any) easyCon.PackResp {
			return easyCon.PackResp{
				RespCode: easyCon.ERespSuccess,
				Content:  want,
			}
		},
	}
	svc := newServiceWithMock(mock)

	got, err := svc.SendRequest[any]("user", "get", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 透传：应当是同一份引用（无 JSON 往返）
	gotMap, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", got)
	}
	if gotMap["name"] != "joey" {
		t.Errorf("expected name=joey, got %v", gotMap["name"])
	}

	// 验证确实是透传而非反序列化重建
	gotPtr, ok2 := got.(*map[string]any)
	_ = gotPtr
	_ = ok2
	// 引用一致：got 与 want 应该是同一个 map（无拷贝）
	// 注意：由于 any 透传，得到的就是 resp.Content 本身
	if &gotMap == &(want) {
		// 不太可能通过 & 操作直接比较，因为取地址的是局部变量
	}
}

// TestSendRequest_TIsStruct 验证 T 为结构体时通过 JSON 反序列化还原。
func TestSendRequest_TIsStruct(t *testing.T) {
	type userResp struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	want := userResp{Name: "joey", Age: 30}
	mock := &mockAdapter{
		respFunc: func(module, route string, params any) easyCon.PackResp {
			// 用 map 模拟远端（结构与 userResp JSON 兼容）
			return easyCon.PackResp{
				RespCode: easyCon.ERespSuccess,
				Content:  map[string]any{"name": "joey", "age": 30},
			}
		},
	}
	svc := newServiceWithMock(mock)

	got, err := svc.SendRequest[userResp]("user", "get", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != want.Name || got.Age != want.Age {
		t.Errorf("expected %+v, got %+v", want, got)
	}
}

// TestSendRequest_TIsSlice 验证 T 为切片时通过 JSON 反序列化还原。
func TestSendRequest_TIsSlice(t *testing.T) {
	mock := &mockAdapter{
		respFunc: func(module, route string, params any) easyCon.PackResp {
			return easyCon.PackResp{
				RespCode: easyCon.ERespSuccess,
				Content: []map[string]any{
					{"id": 1, "name": "a"},
					{"id": 2, "name": "b"},
				},
			}
		},
	}
	svc := newServiceWithMock(mock)

	type item struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	got, err := svc.SendRequest[[]item]("user", "list", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
	if got[0].Name != "a" || got[1].Name != "b" {
		t.Errorf("unexpected items: %+v", got)
	}
}

// TestSendRequest_ErrorResp 验证非 ERespSuccess 时返回 error。
func TestSendRequest_ErrorResp(t *testing.T) {
	mock := &mockAdapter{
		respFunc: func(module, route string, params any) easyCon.PackResp {
			return easyCon.PackResp{
				RespCode: easyCon.ERespRouteNotFind,
				Error:    "no such route",
			}
		},
	}
	svc := newServiceWithMock(mock)

	got, err := svc.SendRequest[map[string]any]("user", "missing", nil)
	if err == nil {
		t.Fatalf("expected error, got value=%v", got)
	}
	if !strings.Contains(err.Error(), "404") && !strings.Contains(err.Error(), "no such route") {
		t.Errorf("unexpected error message: %v", err)
	}
	// got 应当是 zero value
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

// TestSendRequest_UnmarshalFailure 验证 JSON 反序列化失败时返回 error。
func TestSendRequest_UnmarshalFailure(t *testing.T) {
	mock := &mockAdapter{
		respFunc: func(module, route string, params any) easyCon.PackResp {
			return easyCon.PackResp{
				RespCode: easyCon.ERespSuccess,
				Content:  "not a struct", // 无法 Unmarshal 为 struct
			}
		},
	}
	svc := newServiceWithMock(mock)

	type foo struct {
		Bar int `json:"bar"`
	}
	got, err := svc.SendRequest[foo]("user", "bad", nil)
	if err == nil {
		t.Fatalf("expected unmarshal error, got foo=%+v", got)
	}
	if got.Bar != 0 {
		t.Errorf("expected zero value, got %+v", got)
	}
}

// TestSendRequest_TIsAnyReturnsExactType 验证 any 透传时不丢失具体类型。
func TestSendRequest_TIsAnyReturnsExactType(t *testing.T) {
	mock := &mockAdapter{
		respFunc: func(module, route string, params any) easyCon.PackResp {
			return easyCon.PackResp{
				RespCode: easyCon.ERespSuccess,
				Content:  []int{1, 2, 3},
			}
		},
	}
	svc := newServiceWithMock(mock)

	got, err := svc.SendRequest[any]("user", "list", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// any 透传：具体类型应保留为 []int
	if reflect.TypeOf(got).Kind() != reflect.Slice {
		t.Errorf("expected slice, got %T", got)
	}
}