package artifact

import (
	"testing"
)

func TestType_String(t *testing.T) {
	tests := []struct {
		name string
		typ  Type
		want string
	}{
		{"agent", Agent, "agent"},
		{"full", Full, "full"},
		{"unknown returns unknown", Type(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.typ.String(); got != tt.want {
				t.Errorf("Type.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMode(t *testing.T) {
	tests := []struct {
		name string
		full bool
		want Type
	}{
		{"false returns Agent", false, Agent},
		{"true returns Full", true, Full},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Mode(tt.full); got != tt.want {
				t.Errorf("Mode(%v) = %v, want %v", tt.full, got, tt.want)
			}
		})
	}
}

func TestType_IsAgent(t *testing.T) {
	if !Agent.IsAgent() {
		t.Error("Agent.IsAgent() = false, want true")
	}
	if Full.IsAgent() {
		t.Error("Full.IsAgent() = true, want false")
	}
}

func TestType_IsFull(t *testing.T) {
	if !Full.IsFull() {
		t.Error("Full.IsFull() = false, want true")
	}
	if Agent.IsFull() {
		t.Error("Agent.IsFull() = true, want false")
	}
}

func TestNewListResult(t *testing.T) {
	t.Run("with items", func(t *testing.T) {
		items := []string{"a", "b", "c"}
		result := NewListResult(items, true)

		if result.Meta.Count != 3 {
			t.Errorf("Meta.Count = %d, want 3", result.Meta.Count)
		}
		if !result.Meta.HasMore {
			t.Error("Meta.HasMore = false, want true")
		}
		if slice, ok := result.Results.([]string); !ok || len(slice) != 3 {
			t.Error("Results not preserved correctly")
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		items := []int{}
		result := NewListResult(items, false)

		if result.Meta.Count != 0 {
			t.Errorf("Meta.Count = %d, want 0", result.Meta.Count)
		}
		if result.Meta.HasMore {
			t.Error("Meta.HasMore = true, want false")
		}
	})

	t.Run("with struct items", func(t *testing.T) {
		type testItem struct {
			ID   string
			Name string
		}
		items := []testItem{
			{ID: "1", Name: "one"},
			{ID: "2", Name: "two"},
		}
		result := NewListResult(items, false)

		if result.Meta.Count != 2 {
			t.Errorf("Meta.Count = %d, want 2", result.Meta.Count)
		}
	})

	t.Run("nil slice normalized to empty", func(t *testing.T) {
		var items []string // nil slice
		result := NewListResult(items, false)

		if result.Meta.Count != 0 {
			t.Errorf("Meta.Count = %d, want 0", result.Meta.Count)
		}
		// Verify Results is not nil (would serialize as null)
		if result.Results == nil {
			t.Error("Results should not be nil after normalization")
		}
	})
}
