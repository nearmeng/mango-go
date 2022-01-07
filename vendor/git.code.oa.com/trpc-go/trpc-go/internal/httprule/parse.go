package httprule

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

const (
	invalidChar = byte(0)
)

var (
	errParserInternal   = errors.New("parser internal error")
	errEmptyLiteral     = errors.New("empty literal is not allowed")
	errInitialCharAlpha = errors.New("initial char of identifier not alpha")
	errEmptyIdent       = errors.New("empty identifier")
	errNestedVar        = errors.New("nested variables are not allowed")
	errDeepWildcard     = errors.New("deep wildcard must be the last segment")
	errDupFieldPath     = errors.New("dup field path")
	errLeadingSlash     = errors.New("leading slash required")
)

// parser 模板解析器
type parser struct {
	urlPath string // 完整的 httprule url path
	curr    int    // 当前指针
}

// Parse 把 httprule url path 解析到模板中
func Parse(urlPath string) (*PathTemplate, error) {
	p := &parser{
		urlPath: urlPath,
	}

	tpl, err := p.parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse url path %s to template: %w, curr: %d", urlPath, err, p.curr)
	}

	return tpl, nil
}

// parse 开始解析
func (p *parser) parse() (*PathTemplate, error) {
	// 必须以 '/' 开始
	if err := p.consume('/'); err != nil {
		return nil, err
	}

	// 解析 segments 部分
	segments, err := p.parseSegments()
	if err != nil {
		return nil, err
	}
	// 解析 verb 部分
	var verb string
	// 如果最后一个 segment 为 literal 类型，则 verb 已经被包含进去了，找 literal 类型里面最后一个 ':' 位置
	lastSegment := segments[len(segments)-1]
	if lastSegment.kind() == kindLiteral {
		s := lastSegment.String()
		idx := strings.LastIndex(s, ":")
		if idx > 0 {
			verb = s[idx+1:]
			segments[len(segments)-1] = literal(s[:idx])
		}
	} else {
		if err := p.consume(':'); err == nil {
			verb, err = p.parseVerb()
			if err != nil {
				return nil, err
			}
		}
	}

	// 检查是否解析完成
	if !p.done() {
		return nil, errParserInternal
	}

	// 验证
	tpl := &PathTemplate{
		segments: segments,
		verb:     verb,
	}
	if err := p.validate(tpl); err != nil {
		return nil, err
	}

	return tpl, nil
}

// validate 验证 template，两个方面：
// 1. 是否有内嵌变量
// 2. ** 是否是最后一个 segment
// 3. 检查是否存在重复变量名
func (p *parser) validate(tpl *PathTemplate) error {
	m := make(map[string]bool) // 记录重复变量名

	for i, segment := range tpl.segments {
		// 如果是变量类型，先检查重复变量名，再检查其嵌套 segments：
		// 1. 是否有嵌套变量
		// 2. 如果 i != len(tpl.segments) - 1，则嵌套变量中不能有 **
		// 3. 如果 i == len(tpl.segments) - 1，则 ** 只能是最后一个嵌套变量
		if segment.kind() == kindVariable {
			// 重复变量名
			s := strings.Join(segment.fieldPath(), ".")
			if m[s] {
				return errDupFieldPath
			}
			m[s] = true

			// 检查嵌套 segments
			nestedSegments := segment.nestedSegments()
			for j, nestedSegment := range nestedSegments {
				// 嵌套变量
				if nestedSegment.kind() == kindVariable {
					return errNestedVar
				}

				// 如果 i != len(tpl.segments) - 1，则嵌套变量中不能有 **
				if i != len(tpl.segments)-1 && nestedSegment.kind() == kindDeepWildcard {
					return errDeepWildcard
				}

				// 如果 i == len(tpl.segments) - 1，则 ** 只能是最后一个嵌套变量
				if i == len(tpl.segments)-1 && j != len(nestedSegments)-1 &&
					nestedSegment.kind() == kindDeepWildcard {
					return errDeepWildcard
				}
			}
		}

		// 非最后一个 segment 如果是 **，非法
		if i != len(tpl.segments)-1 && segment.kind() == kindDeepWildcard {
			return errDeepWildcard
		}
	}

	return nil
}

// parseSegments 解析 segments
func (p *parser) parseSegments() ([]segment, error) {
	// 至少有一个 segment
	seg, err := p.parseSegment()
	if err != nil {
		return nil, err
	}

	result := []segment{seg}

	if err := p.consume('/'); err == nil {
		// 递归解析 segments
		segs, err := p.parseSegments()
		if err != nil {
			return nil, err
		}
		result = append(result, segs...)
	}

	return result, nil
}

// parseVerb 解析 verb
func (p *parser) parseVerb() (string, error) {
	return p.parseLiteral()
}

// parseSegment 解析单个 segment
func (p *parser) parseSegment() (segment, error) {
	switch p.currentChar() {
	case invalidChar:
		return nil, errParserInternal
	case '*':
		if p.peekN(1) == '*' {
			p.curr++
			p.curr++
			return deepWildcard{}, nil
		}
		p.curr++
		return wildcard{}, nil
	case '{':
		return p.parseVariableSegment()
	default:
		return p.parseLiteralSegment()
	}
}

// parseLiteral 解析 literal 类型
// https://www.ietf.org/rfc/rfc3986.txt, P.49
//   pchar         = unreserved / pct-encoded / sub-delims / ":" / "@"
//   unreserved    = ALPHA / DIGIT / "-" / "." / "_" / "~"
//   sub-delims    = "!" / "$" / "&" / "'" / "(" / ")"
//                 / "*" / "+" / "," / ";" / "="
//   pct-encoded   = "%" HEXDIG HEXDIG
func (p *parser) parseLiteral() (string, error) {
	lit := bytes.Buffer{}

	for {
		// pchar = unreserved / pct-encoded / sub-delims / ":" / "@"
		if isUnreserved(rune(p.currentChar())) || isSubDelims(rune(p.currentChar())) ||
			p.currentChar() == '@' || p.currentChar() == ':' {
			lit.WriteByte(p.currentChar())
			p.curr++
			continue
		} else if isPCTEncoded(rune(p.currentChar()), rune(p.peekN(1)), rune(p.peekN(2))) {
			lit.WriteByte(p.currentChar())
			p.curr++
			lit.WriteByte(p.currentChar())
			p.curr++
			lit.WriteByte(p.currentChar())
			p.curr++
			continue
		} else {
			break
		}
	}

	// 空 literal
	if lit.Len() == 0 {
		return "", errEmptyLiteral
	}

	return lit.String(), nil
}

// parseLiteralSegment 解析 literal segment
func (p *parser) parseLiteralSegment() (segment, error) {
	lit, err := p.parseLiteral()
	if err != nil {
		return nil, err
	}
	return literal(lit), nil
}

// parseVariableSegment 解析 variable segment
func (p *parser) parseVariableSegment() (segment, error) {
	var v variable

	// 变量必须以 '{' 开始
	if err := p.consume('{'); err != nil {
		return nil, err
	}

	// 解析 fieldPath
	fieldPath, err := p.parseFieldPath()
	if err != nil {
		return nil, err
	}
	v.fp = fieldPath

	// 判断是否有 segments
	if err := p.consume('='); err == nil {
		segments, err := p.parseSegments()
		if err != nil {
			return nil, err
		}
		v.segments = segments
	} else { // 没有 segments，则默认通配符 *
		v.segments = []segment{wildcard{}}
	}

	// 变量必须以 '}' 开始
	if err := p.consume('}'); err != nil {
		return nil, err
	}

	return v, nil
}

// parseFieldPath 解析 field path
func (p *parser) parseFieldPath() ([]string, error) {
	// 至少要有一个 ident
	ident, err := p.parseIdent()
	if err != nil {
		return nil, err
	}

	result := []string{ident}

	if err := p.consume('.'); err == nil {
		// 递归解析 fieldPath
		fp, err := p.parseFieldPath()
		if err != nil {
			return nil, err
		}
		result = append(result, fp...)
	}
	return result, nil
}

// parseIdent 解析 ident, 有效的 ident 格式为 ([[:alpha:]_][[:alphanum:]_]*)
func (p *parser) parseIdent() (string, error) {
	ident := bytes.Buffer{}

	for {
		if ident.Len() == 0 && !isAlpha(rune(p.currentChar())) {
			return "", errInitialCharAlpha
		}
		if isAlpha(rune(p.currentChar())) || isDigit(rune(p.currentChar())) || p.currentChar() == '_' {
			ident.WriteByte(p.currentChar())
			p.curr++
			continue
		}
		break
	}

	// 空 ident
	if ident.Len() == 0 {
		return "", errEmptyIdent
	}
	return ident.String(), nil
}

// 解析完成
func (p *parser) done() bool {
	return p.curr >= len(p.urlPath)
}

// 当前字符
func (p *parser) currentChar() byte {
	if p.done() {
		return invalidChar
	}
	return p.urlPath[p.curr]
}

// 消费指定字符
func (p *parser) consume(c byte) error {
	if p.currentChar() == c {
		p.curr++
		return nil
	}
	return fmt.Errorf("failed to consume `%c`", c)
}

// 获取从 record 再往前 n 位置的字符
func (p *parser) peekN(n int) byte {
	peekIdx := p.curr + n
	if peekIdx < len(p.urlPath) {
		return p.urlPath[peekIdx]
	}
	return invalidChar
}

// 判断字符是否为 unreserved 类型
func isUnreserved(r rune) bool {
	if isAlpha(r) || isDigit(r) {
		return true
	}
	switch r {
	case '-', '.', '_', '~':
		return true
	default:
		return false
	}
}

// 判断字符是否为字母类型
func isAlpha(r rune) bool {
	return ('A' <= r && r <= 'Z') || ('a' <= r && r <= 'z')
}

// 判断字符是否为数字类型
func isDigit(r rune) bool {
	return '0' <= r && r <= '9'
}

// 判断字符是否为 subDelims类型
func isSubDelims(r rune) bool {
	switch r {
	case '!', '$', '&', '\'', '(', ')', '*', '+', ',', ';', '=':
		return true
	default:
		return false
	}
}

// 判断字符是否为 pctEncoded 类型
func isPCTEncoded(r1, r2, r3 rune) bool {
	return r1 == '%' && isHexDigit(r2) && isHexDigit(r3)
}

// 判断字符是否为 hex 类型
func isHexDigit(r rune) bool {
	switch {
	case '0' <= r && r <= '9':
		return true
	case 'A' <= r && r <= 'F':
		return true
	case 'a' <= r && r <= 'f':
		return true
	default:
		return false
	}
}
