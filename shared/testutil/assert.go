// Package testutil provides lightweight test assertion helpers.
//
// All helpers call t.Helper() so test failures report the caller's line number.
// "Require" variants call t.Fatal (stop the test); plain variants call t.Error
// (mark failed but continue).
package testutil

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// Equal checks that got and want are deeply equal.
func Equal(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// RequireEqual checks that got and want are deeply equal, stopping the test on failure.
func RequireEqual(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// NoError checks that err is nil.
func NoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// RequireNoError checks that err is nil, stopping the test on failure.
func RequireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Error checks that err is not nil.
func Error(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

// RequireError checks that err is not nil, stopping the test on failure.
func RequireError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

// Contains checks that s contains substr.
func Contains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("string %q does not contain %q", s, substr)
	}
}

// True checks that the condition is true.
func True(t *testing.T, condition bool, msgAndArgs ...any) {
	t.Helper()
	if !condition {
		if len(msgAndArgs) > 0 {
			t.Errorf("expected true: %s", fmt.Sprint(msgAndArgs...))
		} else {
			t.Errorf("expected true, got false")
		}
	}
}

// False checks that the condition is false.
func False(t *testing.T, condition bool, msgAndArgs ...any) {
	t.Helper()
	if condition {
		if len(msgAndArgs) > 0 {
			t.Errorf("expected false: %s", fmt.Sprint(msgAndArgs...))
		} else {
			t.Errorf("expected false, got true")
		}
	}
}

// Nil checks that v is nil.
func Nil(t *testing.T, v any) {
	t.Helper()
	if v == nil {
		return
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface ||
		rv.Kind() == reflect.Map || rv.Kind() == reflect.Slice ||
		rv.Kind() == reflect.Chan || rv.Kind() == reflect.Func {
		if rv.IsNil() {
			return
		}
	}
	t.Errorf("expected nil, got %v", v)
}

// NotNil checks that v is not nil.
func NotNil(t *testing.T, v any) {
	t.Helper()
	if v == nil {
		t.Errorf("expected not nil, got nil")
		return
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface ||
		rv.Kind() == reflect.Map || rv.Kind() == reflect.Slice ||
		rv.Kind() == reflect.Chan || rv.Kind() == reflect.Func {
		if rv.IsNil() {
			t.Errorf("expected not nil, got nil")
		}
	}
}

// Len checks that the length of v equals expected.
// v must be a string, slice, array, map, or channel.
func Len(t *testing.T, v any, expected int) {
	t.Helper()
	rv := reflect.ValueOf(v)
	switch rv.Kind() { //nolint:exhaustive // covered by default case
	case reflect.String, reflect.Slice, reflect.Array, reflect.Map, reflect.Chan:
		if rv.Len() != expected {
			t.Errorf("expected length %d, got %d", expected, rv.Len())
		}
	default:
		t.Errorf("Len called on unsupported type %T", v)
	}
}

// Empty checks that v has length 0.
func Empty(t *testing.T, v any) {
	t.Helper()
	rv := reflect.ValueOf(v)
	switch rv.Kind() { //nolint:exhaustive // covered by default case
	case reflect.String, reflect.Slice, reflect.Array, reflect.Map, reflect.Chan:
		if rv.Len() != 0 {
			t.Errorf("expected empty, got length %d", rv.Len())
		}
	default:
		t.Errorf("Empty called on unsupported type %T", v)
	}
}

// NotEmpty checks that v has length > 0.
func NotEmpty(t *testing.T, v any) {
	t.Helper()
	rv := reflect.ValueOf(v)
	switch rv.Kind() { //nolint:exhaustive // covered by default case
	case reflect.String, reflect.Slice, reflect.Array, reflect.Map, reflect.Chan:
		if rv.Len() == 0 {
			t.Errorf("expected not empty")
		}
	default:
		t.Errorf("NotEmpty called on unsupported type %T", v)
	}
}

// Greater checks that a > b (both must be comparable numeric types or strings).
func Greater(t *testing.T, a, b int) {
	t.Helper()
	if a <= b {
		t.Errorf("expected %d > %d", a, b)
	}
}

// GreaterOrEqual checks that a >= b.
func GreaterOrEqual(t *testing.T, a, b int) {
	t.Helper()
	if a < b {
		t.Errorf("expected %d >= %d", a, b)
	}
}

// NotContains checks that s does not contain substr.
func NotContains(t *testing.T, s, substr string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Errorf("string %q should not contain %q", s, substr)
	}
}

// HasPrefix checks that s starts with prefix.
func HasPrefix(t *testing.T, s, prefix string) {
	t.Helper()
	if !strings.HasPrefix(s, prefix) {
		t.Errorf("string %q does not start with %q", s, prefix)
	}
}

// HasSuffix checks that s ends with suffix.
func HasSuffix(t *testing.T, s, suffix string) {
	t.Helper()
	if !strings.HasSuffix(s, suffix) {
		t.Errorf("string %q does not end with %q", s, suffix)
	}
}

// NotEqual checks that got and want are not deeply equal.
func NotEqual(t *testing.T, got, want any) {
	t.Helper()
	if reflect.DeepEqual(got, want) {
		t.Errorf("expected values to differ, both are %v", got)
	}
}

// ErrorContains checks that err is not nil and its message contains substr.
func ErrorContains(t *testing.T, err error, substr string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", substr)
	}
	if !strings.Contains(err.Error(), substr) {
		t.Errorf("error %q does not contain %q", err.Error(), substr)
	}
}
