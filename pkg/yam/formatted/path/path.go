package path

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Note: While YAML allows non-scalar mapping keys, we'll constrain this
// expression system to only account for scalar mapping keys.

type Path struct {
	parts []Part
}

func Root() Path {
	return Path{
		parts: []Part{rootPart{}},
	}
}

const (
	rootExpression   = "."
	anyKeyExpression = "*"
)

var ErrExpressionNotSupported = errors.New("expression not supported")

var (
	regexMapKey   = regexp.MustCompile(`^(\.([0-9a-zA-Z_-]+|\*))([\[.].*)?`)
	regexSeqIndex = regexp.MustCompile(`^\[(\d*)]`)
	regexRootSeq  = regexp.MustCompile(`^\.\[(\d*)]`)
)

func Parse(expression string) (Path, error) {
	if expression == rootExpression {
		return Root(), nil
	}

	result := Root() // i.e., so far

	remaining := expression
	for {
		if remaining == "" {
			return result, nil
		}

		//  check for the expression to specify a sequence index right off the bat, e.g. ".[42]"
		submatches := regexRootSeq.FindStringSubmatch(expression)
		if len(submatches) >= 2 {
			indexString := submatches[1]
			if indexString == "" {
				result = result.AppendSeqPart(anyIndex)
				remaining = strings.TrimPrefix(remaining, submatches[0])
				continue
			}

			index, err := strconv.Atoi(indexString)
			if err != nil {
				return Path{}, ErrExpressionNotSupported
			}
			result = result.AppendSeqPart(index)
			remaining = strings.TrimPrefix(remaining, submatches[0])
			continue
		}

		// check for map key, e.g. ".some-key"
		submatches = regexMapKey.FindStringSubmatch(remaining)
		if len(submatches) >= 3 {
			key := submatches[2]
			if key == anyKeyExpression {
				result = result.AppendMapPart(anyKey)
				remaining = strings.TrimPrefix(remaining, submatches[1])
				continue
			}

			result = result.AppendMapPart(key)
			remaining = strings.TrimPrefix(remaining, submatches[1])
			continue
		}

		// check for seq index, e.g. "[7]"
		submatches = regexSeqIndex.FindStringSubmatch(remaining)
		if len(submatches) >= 2 {
			indexString := submatches[1]
			if indexString == "" {
				result = result.AppendSeqPart(anyIndex)
				remaining = strings.TrimPrefix(remaining, submatches[0])
				continue
			}

			index, err := strconv.Atoi(indexString)
			if err != nil {
				return Path{}, ErrExpressionNotSupported
			}
			result = result.AppendSeqPart(index)
			remaining = strings.TrimPrefix(remaining, submatches[0])
			continue
		}

		// nothing else it could be
		return Path{}, ErrExpressionNotSupported
	}
}

func (p Path) AppendMapPart(key string) Path {
	return Path{
		parts: append(p.parts, mapPart{
			key: key,
		}),
	}
}

func (p Path) AppendSeqPart(index int) Path {
	return Path{
		parts: append(p.parts, seqPart{
			index: index,
		}),
	}
}

func (p Path) Len() int {
	return len(p.parts)
}

func (p Path) Last() Part {
	lastIndex := len(p.parts) - 1
	return p.parts[lastIndex]
}

func (p Path) String() string {
	var result string

	for _, part := range p.parts {
		switch tp := part.(type) {
		case rootPart:
			result = ""
		case mapPart:
			result += fmt.Sprintf(".%s", tp.key)
		case seqPart:
			result += fmt.Sprintf("[%d]", tp.index)
		}
	}

	if result == "" {
		return rootExpression
	}

	return result
}

func (p Path) Matches(testSubject Path) bool {
	if len(p.parts) != len(testSubject.parts) {
		return false
	}

	for i, patternPart := range p.parts {
		if !partsMatch(patternPart, testSubject.parts[i]) {
			return false
		}
	}

	return true
}

func partsMatch(pattern, testSubject Part) bool {
	if pattern.Kind() != testSubject.Kind() {
		return false
	}

	switch tp := pattern.(type) {
	case rootPart:
		return true

	case mapPart:
		ts, _ := testSubject.(mapPart)
		return tp.key == ts.key || tp.key == anyKey

	case seqPart:
		ts, _ := testSubject.(seqPart)
		return tp.index == ts.index || tp.index == anyIndex
	}

	return false
}
