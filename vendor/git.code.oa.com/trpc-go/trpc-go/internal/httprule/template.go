package httprule

import (
	"fmt"
	"strings"
)

// PathTemplate url path 模板：
//
// Template = "/" Segments [ Verb ] ;
// Segments = Segment { "/" Segment } ;
// Segment  = "*" | "**" | LITERAL | Variable ;
// Variable = "{" FieldPath [ "=" Segments ] "}" ;
// FieldPath = IDENT { "." IDENT } ;
// Verb     = ":" LITERAL ;
type PathTemplate struct {
	segments []segment
	verb     string
}

// segment 类型
type segmentKind int

const (
	kindWildcard segmentKind = iota
	kindDeepWildcard
	kindLiteral
	kindVariable
)

// Segment = "*" | "**" | LITERAL | Variable
type segment interface {
	fmt.Stringer
	handle(*matcher) error
	kind() segmentKind
	fieldPath() []string
	nestedSegments() []segment
}

var _ segment = wildcard{}
var _ segment = deepWildcard{}
var _ segment = literal("")
var _ segment = variable{}

// wildcard 表示 *
type wildcard struct{}

// String 实现 segment
func (wildcard) String() string {
	return "*"
}

// fieldPath 实现 segment
func (wildcard) fieldPath() []string {
	return nil
}

// kind 实现 segment
func (wildcard) kind() segmentKind {
	return kindWildcard
}

// nestedSegments 实现 segment
func (wildcard) nestedSegments() []segment {
	return nil
}

// deepWildcard 表示 **
type deepWildcard struct{}

// String 实现 Segment
func (deepWildcard) String() string {
	return "**"
}

// fieldPath 实现 segment
func (deepWildcard) fieldPath() []string {
	return nil
}

// kind 实现 segment
func (deepWildcard) kind() segmentKind {
	return kindDeepWildcard
}

// nestedSegments 实现 segment
func (deepWildcard) nestedSegments() []segment {
	return nil
}

// literal 类似 /foo
type literal string

// String 实现 Segment
func (l literal) String() string {
	return string(l)
}

// fieldPath 实现 segment
func (literal) fieldPath() []string {
	return nil
}

// kind 实现 segment
func (literal) kind() segmentKind {
	return kindLiteral
}

// nestedSegments 实现 segment
func (literal) nestedSegments() []segment {
	return nil
}

// variable 类似 {var=*}，Variable = "{" FieldPath [ "=" Segments ] "}"
type variable struct {
	fp       []string // FieldPath = IDENT { "." IDENT }
	segments []segment
}

// String 实现 segment
func (v variable) String() string {
	ss := make([]string, len(v.segments))
	for i := range v.segments {
		ss[i] = v.segments[i].String()
	}
	return fmt.Sprintf("{%s=%s}", strings.Join(v.fp, "."), strings.Join(ss, "/"))
}

// fieldPath 实现 segment
func (v variable) fieldPath() []string {
	return v.fp
}

// kind 实现 segment
func (variable) kind() segmentKind {
	return kindVariable
}

// nestedSegments 实现 segment
func (v variable) nestedSegments() []segment {
	return v.segments
}

// FieldPaths 获取 field paths
func (tpl *PathTemplate) FieldPaths() [][]string {
	var fps [][]string
	for _, segment := range tpl.segments {
		if fp := segment.fieldPath(); fp != nil {
			fps = append(fps, fp)
		}
	}
	return fps
}
