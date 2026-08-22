package bencode

import (
	"errors"
	"fmt"
	"strconv"
)

// 常见的解析错误。
var (
	ErrUnexpectedEOF = errors.New("bencode: unexpected end of data")
	ErrInvalidData   = errors.New("bencode: invalid data")
)

// Decode 将 bencode 编码的字节流解析为 BNode 树。
//
// 它支持四种 bencode 类型：
//   - 整数:   i<十进制数字>e
//   - 字符串: <十进制长度>:<原始字节>
//   - 列表:   l<元素...>e
//   - 字典:   d<键><值>...e（键必须是 bencode 字符串）
func Decode(data []byte) (*BNode, error) {
	p := &parser{data: data}
	n, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	if p.pos != len(data) {
		return nil, fmt.Errorf("%w: %d trailing bytes", ErrInvalidData, len(data)-p.pos)
	}
	return n, nil
}

// MustDecode 与 Decode 相同，但在出错时 panic。适合解析内嵌的固定结构。
func MustDecode(data []byte) *BNode {
	n, err := Decode(data)
	if err != nil {
		panic(err)
	}
	return n
}

type parser struct {
	data []byte
	pos  int
}

func (p *parser) parseValue() (*BNode, error) {
	if p.pos >= len(p.data) {
		return nil, ErrUnexpectedEOF
	}
	switch c := p.data[p.pos]; c {
	case 'i':
		return p.parseInt()
	case 'l':
		return p.parseList()
	case 'd':
		return p.parseDict()
	default:
		if c >= '0' && c <= '9' {
			return p.parseString()
		}
		return nil, fmt.Errorf("%w: unexpected byte %q at offset %d", ErrInvalidData, c, p.pos)
	}
}

// parseInt 解析 i<数字>e 形式。
func (p *parser) parseInt() (*BNode, error) {
	p.pos++ // 跳过 'i'
	start := p.pos
	for p.pos < len(p.data) && p.data[p.pos] != 'e' {
		p.pos++
	}
	if p.pos >= len(p.data) {
		return nil, ErrUnexpectedEOF
	}
	raw := string(p.data[start:p.pos])
	p.pos++ // 跳过 'e'
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: bad integer %q", ErrInvalidData, raw)
	}
	return &BNode{Type: BInt, Int: v}, nil
}

// parseString 解析 <长度>:<字节> 形式。
func (p *parser) parseString() (*BNode, error) {
	start := p.pos
	for p.pos < len(p.data) && p.data[p.pos] != ':' {
		p.pos++
	}
	if p.pos >= len(p.data) {
		return nil, ErrUnexpectedEOF
	}
	lenStr := string(p.data[start:p.pos])
	n, err := strconv.Atoi(lenStr)
	if err != nil || n < 0 {
		return nil, fmt.Errorf("%w: bad string length %q", ErrInvalidData, lenStr)
	}
	p.pos++ // 跳过 ':'
	if p.pos+n > len(p.data) {
		return nil, ErrUnexpectedEOF
	}
	s := string(p.data[p.pos : p.pos+n])
	p.pos += n
	return &BNode{Type: BString, Str: s}, nil
}

// parseList 解析 l<元素...>e 形式。
func (p *parser) parseList() (*BNode, error) {
	p.pos++ // 跳过 'l'
	node := &BNode{Type: BList}
	for {
		if p.pos >= len(p.data) {
			return nil, ErrUnexpectedEOF
		}
		if p.data[p.pos] == 'e' {
			p.pos++
			return node, nil
		}
		item, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		node.List = append(node.List, item)
	}
}

// parseDict 解析 d<键><值>...e 形式，键必须是 bencode 字符串。
func (p *parser) parseDict() (*BNode, error) {
	p.pos++ // 跳过 'd'
	node := &BNode{Type: BDict, Dict: make(map[string]*BNode)}
	for {
		if p.pos >= len(p.data) {
			return nil, ErrUnexpectedEOF
		}
		if p.data[p.pos] == 'e' {
			p.pos++
			return node, nil
		}
		key, err := p.parseString()
		if err != nil {
			return nil, err
		}
		val, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		node.Dict[key.Str] = val
	}
}

// Get 返回字典中指定键的值；非字典或键不存在时返回 nil。
func (n *BNode) Get(key string) *BNode {
	if n == nil || n.Type != BDict {
		return nil
	}
	return n.Dict[key]
}

// GetString 便捷方法：返回字典键的字符串值；不存在或类型不符时返回空串。
func (n *BNode) GetString(key string) string {
	if v := n.Get(key); v != nil && v.Type == BString {
		return v.Str
	}
	return ""
}

// GetInt 便捷方法：返回字典键的整数值；不存在或类型不符时返回 0。
func (n *BNode) GetInt(key string) int64 {
	if v := n.Get(key); v != nil && v.Type == BInt {
		return v.Int
	}
	return 0
}
