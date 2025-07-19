package ranges

import (
	"fmt"
	"strconv"
	"strings"
)

// Range represents a range of numbers
type Range struct {
	Begin float64
	End   float64
}

// Parse parses a string and returns a slice of ranges
// Supports formats like: "1", "1-5", "1,3,5-10", "1.5-2.5"
func Parse(rnge string) (rngs []Range, err error) {
	if rnge == "" {
		return []Range{}, nil
	}

	co := strings.Split(rnge, ",")

	for _, part := range co {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		in := strings.Split(part, "-")

		// Handle invalid range formats (too many dashes)
		if len(in) > 2 {
			return nil, fmt.Errorf("invalid range format: %s", part)
		}

		// Parse the first number
		beginStr := strings.TrimSpace(in[0])
		begin, err := strconv.ParseFloat(beginStr, 64)
		if err != nil {
			return nil, err
		}

		var end float64
		if len(in) == 2 {
			// This is a range (e.g., "1-5")
			endStr := strings.TrimSpace(in[1])
			end, err = strconv.ParseFloat(endStr, 64)
			if err != nil {
				return nil, err
			}
		} else {
			// This is a single number (e.g., "1")
			end = begin
		}

		// Ensure begin <= end
		if begin > end {
			begin, end = end, begin
		}

		rngs = append(rngs, Range{
			Begin: begin,
			End:   end,
		})
	}

	return rngs, nil
}

// Contains checks if a number is within any of the ranges
func (r Range) Contains(num float64) bool {
	return num >= r.Begin && num <= r.End
}

// ContainsAny checks if a number is within any of the provided ranges
func ContainsAny(ranges []Range, num float64) bool {
	for _, r := range ranges {
		if r.Contains(num) {
			return true
		}
	}
	return false
}

// String returns a string representation of the range
func (r Range) String() string {
	if r.Begin == r.End {
		if r.Begin == float64(int64(r.Begin)) {
			return strconv.FormatFloat(r.Begin, 'f', 0, 64)
		}
		return strconv.FormatFloat(r.Begin, 'f', 1, 64)
	}

	beginStr := strconv.FormatFloat(r.Begin, 'f', -1, 64)
	endStr := strconv.FormatFloat(r.End, 'f', -1, 64)
	return beginStr + "-" + endStr
}

// ToString converts a slice of ranges back to a string representation
func ToString(ranges []Range) string {
	if len(ranges) == 0 {
		return ""
	}

	var parts []string
	for _, r := range ranges {
		parts = append(parts, r.String())
	}

	return strings.Join(parts, ",")
}

// Count returns the total number of individual numbers covered by all ranges
func Count(ranges []Range) int {
	count := 0
	for _, r := range ranges {
		// For simplicity, we count integer ranges
		// For decimal ranges, this might not be as meaningful
		if r.Begin == float64(int64(r.Begin)) && r.End == float64(int64(r.End)) {
			count += int(r.End - r.Begin + 1)
		} else {
			count++ // Count decimal ranges as 1
		}
	}
	return count
}

// Merge combines overlapping ranges into a single range
func Merge(ranges []Range) []Range {
	if len(ranges) == 0 {
		return ranges
	}

	// Sort ranges by begin value
	sorted := make([]Range, len(ranges))
	copy(sorted, ranges)

	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].Begin > sorted[j].Begin {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	merged := []Range{sorted[0]}

	for i := 1; i < len(sorted); i++ {
		current := sorted[i]
		last := &merged[len(merged)-1]

		// If current range overlaps with the last merged range
		if current.Begin <= last.End+1 {
			// Merge them by extending the end if necessary
			if current.End > last.End {
				last.End = current.End
			}
		} else {
			// No overlap, add as new range
			merged = append(merged, current)
		}
	}

	return merged
}
