package ranges

import (
	"reflect"
	"testing"
)

func TestParse_SingleNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Range
	}{
		{
			name:     "single integer",
			input:    "5",
			expected: []Range{{Begin: 5, End: 5}},
		},
		{
			name:     "single decimal",
			input:    "1.5",
			expected: []Range{{Begin: 1.5, End: 1.5}},
		},
		{
			name:     "zero",
			input:    "0",
			expected: []Range{{Begin: 0, End: 0}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Parse() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParse_Range(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Range
	}{
		{
			name:     "simple range",
			input:    "1-5",
			expected: []Range{{Begin: 1, End: 5}},
		},
		{
			name:     "decimal range",
			input:    "1.5-3.5",
			expected: []Range{{Begin: 1.5, End: 3.5}},
		},
		{
			name:     "reverse range (should be corrected)",
			input:    "5-1",
			expected: []Range{{Begin: 1, End: 5}},
		},
		{
			name:     "same number range",
			input:    "3-3",
			expected: []Range{{Begin: 3, End: 3}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Parse() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParse_MultipleRanges(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Range
	}{
		{
			name:     "multiple single numbers",
			input:    "1,3,5",
			expected: []Range{{Begin: 1, End: 1}, {Begin: 3, End: 3}, {Begin: 5, End: 5}},
		},
		{
			name:     "mixed single and range",
			input:    "1,3-5,8",
			expected: []Range{{Begin: 1, End: 1}, {Begin: 3, End: 5}, {Begin: 8, End: 8}},
		},
		{
			name:     "multiple ranges",
			input:    "1-3,5-7,10-12",
			expected: []Range{{Begin: 1, End: 3}, {Begin: 5, End: 7}, {Begin: 10, End: 12}},
		},
		{
			name:     "complex mixed",
			input:    "1,2.5-3.5,5,7-9",
			expected: []Range{{Begin: 1, End: 1}, {Begin: 2.5, End: 3.5}, {Begin: 5, End: 5}, {Begin: 7, End: 9}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Parse() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParse_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Range
		hasError bool
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []Range{},
			hasError: false,
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: []Range{},
			hasError: false,
		},
		{
			name:     "with spaces",
			input:    "1, 3-5, 8",
			expected: []Range{{Begin: 1, End: 1}, {Begin: 3, End: 5}, {Begin: 8, End: 8}},
			hasError: false,
		},
		{
			name:     "trailing comma",
			input:    "1,3,5,",
			expected: []Range{{Begin: 1, End: 1}, {Begin: 3, End: 3}, {Begin: 5, End: 5}},
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(tt.input)
			if tt.hasError && err == nil {
				t.Errorf("Parse() expected error but got none")
			}
			if !tt.hasError && err != nil {
				t.Errorf("Parse() unexpected error = %v", err)
			}
			if !tt.hasError {
				if len(result) == 0 && len(tt.expected) == 0 {
					// Both are empty, this is ok
				} else if !reflect.DeepEqual(result, tt.expected) {
					t.Errorf("Parse() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func TestParse_InvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "invalid number",
			input: "abc",
		},
		{
			name:  "invalid range",
			input: "1-abc",
		},
		{
			name:  "too many dashes",
			input: "1-2-3",
		},
		{
			name:  "invalid character",
			input: "1,@,3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(tt.input)
			if err == nil {
				t.Errorf("Parse() expected error for input %s but got none", tt.input)
			}
			if result != nil {
				t.Errorf("Parse() expected nil result for error case but got %v", result)
			}
		})
	}
}

func TestRange_Contains(t *testing.T) {
	tests := []struct {
		name     string
		r        Range
		num      float64
		expected bool
	}{
		{
			name:     "number in range",
			r:        Range{Begin: 1, End: 5},
			num:      3,
			expected: true,
		},
		{
			name:     "number at begin",
			r:        Range{Begin: 1, End: 5},
			num:      1,
			expected: true,
		},
		{
			name:     "number at end",
			r:        Range{Begin: 1, End: 5},
			num:      5,
			expected: true,
		},
		{
			name:     "number below range",
			r:        Range{Begin: 1, End: 5},
			num:      0,
			expected: false,
		},
		{
			name:     "number above range",
			r:        Range{Begin: 1, End: 5},
			num:      6,
			expected: false,
		},
		{
			name:     "decimal in decimal range",
			r:        Range{Begin: 1.5, End: 3.5},
			num:      2.7,
			expected: true,
		},
		{
			name:     "single number range",
			r:        Range{Begin: 5, End: 5},
			num:      5,
			expected: true,
		},
		{
			name:     "single number range miss",
			r:        Range{Begin: 5, End: 5},
			num:      4,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.r.Contains(tt.num)
			if result != tt.expected {
				t.Errorf("Range.Contains() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestContainsAny(t *testing.T) {
	ranges := []Range{
		{Begin: 1, End: 3},
		{Begin: 5, End: 7},
		{Begin: 10, End: 10},
	}

	tests := []struct {
		name     string
		num      float64
		expected bool
	}{
		{
			name:     "in first range",
			num:      2,
			expected: true,
		},
		{
			name:     "in second range",
			num:      6,
			expected: true,
		},
		{
			name:     "in third range",
			num:      10,
			expected: true,
		},
		{
			name:     "not in any range",
			num:      4,
			expected: false,
		},
		{
			name:     "not in any range (above all)",
			num:      15,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsAny(ranges, tt.num)
			if result != tt.expected {
				t.Errorf("ContainsAny() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRange_String(t *testing.T) {
	tests := []struct {
		name     string
		r        Range
		expected string
	}{
		{
			name:     "single integer",
			r:        Range{Begin: 5, End: 5},
			expected: "5",
		},
		{
			name:     "single decimal",
			r:        Range{Begin: 1.5, End: 1.5},
			expected: "1.5",
		},
		{
			name:     "integer range",
			r:        Range{Begin: 1, End: 5},
			expected: "1-5",
		},
		{
			name:     "decimal range",
			r:        Range{Begin: 1.5, End: 3.7},
			expected: "1.5-3.7",
		},
		{
			name:     "mixed range",
			r:        Range{Begin: 1, End: 3.5},
			expected: "1-3.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.r.String()
			if result != tt.expected {
				t.Errorf("Range.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		name     string
		ranges   []Range
		expected string
	}{
		{
			name:     "empty ranges",
			ranges:   []Range{},
			expected: "",
		},
		{
			name:     "single range",
			ranges:   []Range{{Begin: 1, End: 5}},
			expected: "1-5",
		},
		{
			name:     "multiple ranges",
			ranges:   []Range{{Begin: 1, End: 3}, {Begin: 5, End: 5}, {Begin: 7, End: 9}},
			expected: "1-3,5,7-9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToString(tt.ranges)
			if result != tt.expected {
				t.Errorf("ToString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCount(t *testing.T) {
	tests := []struct {
		name     string
		ranges   []Range
		expected int
	}{
		{
			name:     "empty ranges",
			ranges:   []Range{},
			expected: 0,
		},
		{
			name:     "single number",
			ranges:   []Range{{Begin: 5, End: 5}},
			expected: 1,
		},
		{
			name:     "simple range",
			ranges:   []Range{{Begin: 1, End: 5}},
			expected: 5,
		},
		{
			name:     "multiple ranges",
			ranges:   []Range{{Begin: 1, End: 3}, {Begin: 5, End: 7}},
			expected: 6,
		},
		{
			name:     "decimal range (counted as 1)",
			ranges:   []Range{{Begin: 1.5, End: 3.7}},
			expected: 1,
		},
		{
			name:     "mixed integer and decimal",
			ranges:   []Range{{Begin: 1, End: 3}, {Begin: 5.5, End: 7.2}},
			expected: 4, // 3 integers + 1 decimal range
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Count(tt.ranges)
			if result != tt.expected {
				t.Errorf("Count() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMerge(t *testing.T) {
	tests := []struct {
		name     string
		ranges   []Range
		expected []Range
	}{
		{
			name:     "empty ranges",
			ranges:   []Range{},
			expected: []Range{},
		},
		{
			name:     "single range",
			ranges:   []Range{{Begin: 1, End: 5}},
			expected: []Range{{Begin: 1, End: 5}},
		},
		{
			name:     "no overlap",
			ranges:   []Range{{Begin: 1, End: 3}, {Begin: 5, End: 7}},
			expected: []Range{{Begin: 1, End: 3}, {Begin: 5, End: 7}},
		},
		{
			name:     "overlapping ranges",
			ranges:   []Range{{Begin: 1, End: 5}, {Begin: 3, End: 8}},
			expected: []Range{{Begin: 1, End: 8}},
		},
		{
			name:     "adjacent ranges",
			ranges:   []Range{{Begin: 1, End: 3}, {Begin: 4, End: 6}},
			expected: []Range{{Begin: 1, End: 6}},
		},
		{
			name:     "multiple overlaps",
			ranges:   []Range{{Begin: 1, End: 3}, {Begin: 2, End: 5}, {Begin: 4, End: 7}},
			expected: []Range{{Begin: 1, End: 7}},
		},
		{
			name:     "unsorted input",
			ranges:   []Range{{Begin: 5, End: 7}, {Begin: 1, End: 3}, {Begin: 2, End: 4}},
			expected: []Range{{Begin: 1, End: 7}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Merge(tt.ranges)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Merge() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseAndFilter_Integration(t *testing.T) {
	// Integration test: parse a complex range string and test filtering
	rangeStr := "1,3-5,7.5,10-12"
	ranges, err := Parse(rangeStr)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	testNumbers := []float64{0, 1, 2, 3, 4, 5, 6, 7, 7.5, 8, 9, 10, 11, 12, 13}
	expectedInRange := []bool{false, true, false, true, true, true, false, false, true, false, false, true, true, true, false}

	for i, num := range testNumbers {
		result := ContainsAny(ranges, num)
		if result != expectedInRange[i] {
			t.Errorf("Number %v: ContainsAny() = %v, want %v", num, result, expectedInRange[i])
		}
	}
}
