package overlay

import (
	"fmt"
	"strconv"
	"strings"
)

// matcher decides whether an element matches a target selector.
type matcher interface {
	match(el map[string]any) bool
}

// compile parses a target selector into a matcher. It supports:
//   - selector sugar: "byName:Order", "byType:PartDefinition"
//   - whole-array selectors: "$", "$[*]", "$..*"
//   - JSONPath filter: "$[?(<expr>)]"
func compile(target string) (matcher, error) {
	t := strings.TrimSpace(target)
	switch {
	case t == "":
		return nil, fmt.Errorf("empty target")
	case strings.HasPrefix(t, "byName:"):
		return fieldEq{key: "declaredName", val: strings.TrimPrefix(t, "byName:")}, nil
	case strings.HasPrefix(t, "byType:"):
		return fieldEq{key: "@type", val: strings.TrimPrefix(t, "byType:")}, nil
	case t == "$", t == "$[*]", t == "$..*", t == "$.*":
		return matchAll{}, nil
	}
	if strings.HasPrefix(t, "$[?(") && strings.HasSuffix(t, ")]") {
		expr := t[len("$[?(") : len(t)-len(")]")]
		return parseExpr(expr)
	}
	return nil, fmt.Errorf("unsupported selector %q", target)
}

// matchAll matches every element.
type matchAll struct{}

func (matchAll) match(map[string]any) bool { return true }

// fieldEq is the selector-sugar equality matcher.
type fieldEq struct{ key, val string }

func (f fieldEq) match(el map[string]any) bool {
	s, _ := el[f.key].(string)
	return s == f.val
}

// orMatcher / andMatcher compose comparison clauses.
type orMatcher []matcher

func (o orMatcher) match(el map[string]any) bool {
	for _, m := range o {
		if m.match(el) {
			return true
		}
	}
	return false
}

type andMatcher []matcher

func (a andMatcher) match(el map[string]any) bool {
	for _, m := range a {
		if !m.match(el) {
			return false
		}
	}
	return true
}

// comparison is a single `@.field OP value` (or existence) test.
type comparison struct {
	field string
	op    string // "==", "!=", "exists"
	value any
}

func (c comparison) match(el map[string]any) bool {
	v, ok := el[c.field]
	switch c.op {
	case "exists":
		return ok && truthy(v)
	case "==":
		return ok && equalVal(v, c.value)
	case "!=":
		return !ok || !equalVal(v, c.value)
	}
	return false
}

// parseExpr parses an expression of `&&`/`||`-joined comparisons.
func parseExpr(expr string) (matcher, error) {
	var ors orMatcher
	for _, orPart := range splitTop(expr, "||") {
		var ands andMatcher
		for _, andPart := range splitTop(orPart, "&&") {
			cmp, err := parseComparison(strings.TrimSpace(andPart))
			if err != nil {
				return nil, err
			}
			ands = append(ands, cmp)
		}
		if len(ands) == 1 {
			ors = append(ors, ands[0])
		} else {
			ors = append(ors, ands)
		}
	}
	if len(ors) == 1 {
		return ors[0], nil
	}
	return ors, nil
}

func parseComparison(s string) (comparison, error) {
	op := ""
	if strings.Contains(s, "!=") {
		op = "!="
	} else if strings.Contains(s, "==") {
		op = "=="
	}
	if op == "" {
		// Existence/truthy test: @.field or @['field'].
		field, err := parseFieldRef(strings.TrimSpace(s))
		if err != nil {
			return comparison{}, err
		}
		return comparison{field: field, op: "exists"}, nil
	}
	idx := strings.Index(s, op)
	lhs := strings.TrimSpace(s[:idx])
	rhs := strings.TrimSpace(s[idx+len(op):])
	field, err := parseFieldRef(lhs)
	if err != nil {
		return comparison{}, err
	}
	return comparison{field: field, op: op, value: parseLiteral(rhs)}, nil
}

// parseFieldRef extracts the field name from `@.field` or `@['field']`.
func parseFieldRef(s string) (string, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "@")
	switch {
	case strings.HasPrefix(s, "."):
		return strings.TrimSpace(s[1:]), nil
	case strings.HasPrefix(s, "['") && strings.HasSuffix(s, "']"):
		return s[2 : len(s)-2], nil
	case strings.HasPrefix(s, "[\"") && strings.HasSuffix(s, "\"]"):
		return s[2 : len(s)-2], nil
	}
	return "", fmt.Errorf("invalid field reference %q", s)
}

func parseLiteral(s string) any {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && (s[0] == '\'' || s[0] == '"') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	switch s {
	case "true":
		return true
	case "false":
		return false
	case "null":
		return nil
	}
	if n, err := strconv.ParseFloat(s, 64); err == nil {
		return n
	}
	return s
}

func equalVal(a, b any) bool {
	switch bv := b.(type) {
	case string:
		av, ok := a.(string)
		return ok && av == bv
	case bool:
		av, ok := a.(bool)
		return ok && av == bv
	case float64:
		switch av := a.(type) {
		case float64:
			return av == bv
		case int:
			return float64(av) == bv
		}
	case nil:
		return a == nil
	}
	return fmt.Sprint(a) == fmt.Sprint(b)
}

func truthy(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t != ""
	case nil:
		return false
	default:
		return true
	}
}

// splitTop splits s on sep, ignoring sep occurrences inside quotes.
func splitTop(s, sep string) []string {
	var parts []string
	var depth int
	var inSingle, inDouble bool
	start := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\'' && !inDouble:
			inSingle = !inSingle
		case c == '"' && !inSingle:
			inDouble = !inDouble
		case c == '(' && !inSingle && !inDouble:
			depth++
		case c == ')' && !inSingle && !inDouble:
			depth--
		}
		if depth == 0 && !inSingle && !inDouble && strings.HasPrefix(s[i:], sep) {
			parts = append(parts, s[start:i])
			i += len(sep) - 1
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}
