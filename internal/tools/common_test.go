package tools

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func textOf(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) == 0 {
		t.Fatal("expected at least one content entry")
	}
	tc, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", res.Content[0])
	}
	return tc.Text
}

func makeRequest(args map[string]any) mcp.CallToolRequest {
	req := mcp.CallToolRequest{}
	if args != nil {
		req.Params.Arguments = args
	}
	return req
}

func TestJSONResult(t *testing.T) {
	res := JSONResult([]byte(`{"ok":true}`))
	if res.IsError {
		t.Fatal("expected non-error result")
	}
	if got := textOf(t, res); got != `{"ok":true}` {
		t.Fatalf("unexpected text: %q", got)
	}
}

func TestAPIError(t *testing.T) {
	res := APIError(errors.New("boom"))
	if !res.IsError {
		t.Fatal("expected IsError=true")
	}
	got := textOf(t, res)
	if !strings.HasPrefix(got, "HubSpot API error: ") {
		t.Fatalf("expected HubSpot API error prefix, got %q", got)
	}
	if !strings.Contains(got, "boom") {
		t.Fatalf("expected underlying error message, got %q", got)
	}
}

func TestRequiredStringArg(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]any
		key     string
		want    string
		wantErr bool
	}{
		{name: "present", args: map[string]any{"id": "abc"}, key: "id", want: "abc"},
		{name: "missing", args: map[string]any{}, key: "id", wantErr: true},
		{name: "wrong-type", args: map[string]any{"id": 42}, key: "id", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RequiredStringArg(makeRequest(tc.args), tc.key)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error mismatch: got %v, wantErr=%v", err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestOptionalStringArg(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
		def  string
		want string
	}{
		{name: "present", args: map[string]any{"k": "v"}, def: "x", want: "v"},
		{name: "missing-uses-default", args: map[string]any{}, def: "fallback", want: "fallback"},
		{name: "wrong-type-uses-default", args: map[string]any{"k": 5}, def: "fallback", want: "fallback"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := OptionalStringArg(makeRequest(tc.args), "k", tc.def); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestOptionalIntArg(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
		def  int
		want int
	}{
		{name: "int", args: map[string]any{"n": 7}, def: 0, want: 7},
		{name: "float64-from-json", args: map[string]any{"n": float64(11)}, def: 0, want: 11},
		{name: "string-numeric", args: map[string]any{"n": "13"}, def: 0, want: 13},
		{name: "missing-uses-default", args: map[string]any{}, def: 99, want: 99},
		{name: "wrong-type-uses-default", args: map[string]any{"n": []any{1}}, def: 5, want: 5},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := OptionalIntArg(makeRequest(tc.args), "n", tc.def); got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestOptionalStringArrayArg(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
		want []string
	}{
		{name: "absent-returns-nil", args: map[string]any{}, want: nil},
		{name: "nil-value-returns-nil", args: map[string]any{"k": nil}, want: nil},
		{name: "any-slice-of-strings", args: map[string]any{"k": []any{"a", "b"}}, want: []string{"a", "b"}},
		{name: "string-slice", args: map[string]any{"k": []string{"x", "y"}}, want: []string{"x", "y"}},
		{name: "any-slice-with-nonstring-returns-nil", args: map[string]any{"k": []any{"a", 5}}, want: nil},
		{name: "wrong-type-returns-nil", args: map[string]any{"k": "not-a-slice"}, want: nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := OptionalStringArrayArg(makeRequest(tc.args), "k")
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestOptionalObjectArg(t *testing.T) {
	t.Run("absent-returns-nil-nil", func(t *testing.T) {
		got, err := OptionalObjectArg(makeRequest(map[string]any{}), "k")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Fatalf("expected nil map, got %v", got)
		}
	})
	t.Run("present-object", func(t *testing.T) {
		obj := map[string]any{"name": "Acme"}
		got, err := OptionalObjectArg(makeRequest(map[string]any{"k": obj}), "k")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(got, obj) {
			t.Fatalf("got %v, want %v", got, obj)
		}
	})
	t.Run("present-but-wrong-type-errors", func(t *testing.T) {
		got, err := OptionalObjectArg(makeRequest(map[string]any{"k": "scalar"}), "k")
		if err == nil {
			t.Fatal("expected error for non-object value")
		}
		if got != nil {
			t.Fatalf("expected nil map, got %v", got)
		}
	})
	t.Run("nil-value-treated-as-absent", func(t *testing.T) {
		got, err := OptionalObjectArg(makeRequest(map[string]any{"k": nil}), "k")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Fatalf("expected nil map, got %v", got)
		}
	})
}
