package bencode

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
)

// 常见的解析错误。
var (
	ErrUnexpectedEOF      = errors.New("bencode: unexpected end of data")
	ErrInvalidData        = errors.New("bencode: invalid data")
	ErrUnexpectedTypeData = errors.New("bencode: 从错误的类型中无法获取到值")
)

// Decode 完整解析bencode，必须消费全部字节，尾部多余字节返回错误
func Decode(data []byte) (*BNode, error) {
	pos := 0
	n, err := DecodeValue(data, &pos)
	if err != nil {
		return nil, err
	}
	if pos > len(data) {
		return nil, fmt.Errorf("%w: %d trailing bytes", ErrInvalidData, len(data)-pos)
	}
	return n, nil
}

func DecodeValue(data []byte, pos *int) (*BNode, error) {
	i := *pos
	if i >= len(data) {
		return nil, ErrUnexpectedEOF
	}
	switch c := data[i]; c {
	case 'i':
		return parseInt(data, pos)
	case 'l':
		return parseList(data, pos)
	case 'd':
		return parseDict(data, pos)
	default:
		if c >= '0' && c <= '9' {
			return parseString(data, pos)
		}
		return nil, fmt.Errorf("%w: unexpected byte %q at offset %d", ErrInvalidData, c, *pos)
	}
}

// parseInt 解析 i<数字>e 形式。
func parseInt(data []byte, pos *int) (*BNode, error) {
	i := *pos
	i++ // 跳过 'i'
	start := i
	for i < len(data) && data[i] != 'e' {
		i++
	}
	if i >= len(data) {
		return nil, ErrUnexpectedEOF
	}
	raw := string(data[start:i])
	// bencode规范禁止 -0
	if raw == "-0" {
		return nil, fmt.Errorf("%w: invalid integer -0", ErrInvalidData)
	}
	i++ // 跳过 'e'
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: bad integer %q", ErrInvalidData, raw)
	}
	*pos = i //写回外部的指针, 改变外部的指针对应的值
	return &BNode{Type: BInt, Int: int(v), Raw: data[start-1 : i]}, nil
}

// parseString 解析 <长度>:<字节> 形式。
func parseString(data []byte, pos *int) (*BNode, error) {
	i := *pos
	start := i
	for i < len(data) && data[i] != ':' {
		i++
	}
	if i >= len(data) {
		return nil, ErrUnexpectedEOF
	}
	lenStr := string(data[start:i])
	n, err := strconv.Atoi(lenStr)
	if err != nil || n < 0 {
		return nil, fmt.Errorf("%w: bad string length %q", ErrInvalidData, lenStr)
	}
	i++ // 跳过 ':'
	if i+n > len(data) {
		return nil, ErrUnexpectedEOF
	}
	s := data[i : i+n]
	i += n
	*pos = i
	return &BNode{Type: BString, Str: s, Raw: data[start:i]}, nil
}

// parseList 解析 l<元素...>e 形式。
func parseList(data []byte, pos *int) (*BNode, error) {
	*pos++ // 跳过 'l'
	node := &BNode{Type: BList}
	for {
		if *pos >= len(data) {
			return nil, ErrUnexpectedEOF
		}
		if data[*pos] == 'e' {
			*pos++
			return node, nil
		}
		item, err := DecodeValue(data, pos)
		if err != nil {
			return nil, err
		}
		node.List = append(node.List, item)
	}
}

// parseDict 解析 d<键><值>...e 形式，键必须是 bencode 字符串。
func parseDict(data []byte, pos *int) (*BNode, error) {
	*pos++ // 跳过 'd'
	node := &BNode{Type: BDict, Dict: make(map[string]*BNode)}
	for {
		if *pos >= len(data) {
			return nil, ErrUnexpectedEOF
		}
		if data[*pos] == 'e' {
			*pos++
			return node, nil
		}
		key, err := parseString(data, pos)
		if err != nil {
			return nil, err
		}
		val, err := DecodeValue(data, pos)
		if err != nil {
			return nil, err
		}
		keyStr, err := key.ToString()
		if err != nil {
			return nil, err
		}
		node.Dict[keyStr] = val
		node.DictKeys = append(node.DictKeys, keyStr) //记录key，编码使用
	}
}

func (n *BNode) ToInt() (int, error) {
	if n == nil || n.Type != BInt {
		return 0, fmt.Errorf("%w: expect Bnode type %s", ErrUnexpectedTypeData, BInt)
	}
	return n.Int, nil
}

func (n *BNode) ToHexString() (string, error) {
	if n == nil || n.Type != BString {
		return "", fmt.Errorf("%w: expect Bnode type %s", ErrUnexpectedTypeData, BString)
	}
	return hex.EncodeToString(n.Str), nil
}
func (n *BNode) ToByteString() ([]byte, error) {
	if n == nil || n.Type != BString {
		return nil, fmt.Errorf("%w: expect Bnode type %s", ErrUnexpectedTypeData, BString)
	}
	return n.Str, nil
}

func (n *BNode) ToString() (string, error) {
	if n == nil || n.Type != BString {
		return "", fmt.Errorf("%w: expect Bnode type %s", ErrUnexpectedTypeData, BString)
	}
	return string(n.Str), nil
}

func (n *BNode) ToList() ([]*BNode, error) {
	if n == nil || n.Type != BList {
		return nil, fmt.Errorf("%w: expect Bnode type %s", ErrUnexpectedTypeData, BList)
	}
	return n.List, nil
}

// GetDictValue 返回字典中指定键的值；非字典或键不存在时返回 nil。
func (n *BNode) GetDictValue(key string) (*BNode, error) {
	if n == nil || n.Type != BDict {
		return nil, fmt.Errorf("%w: expect Bnode type %s", ErrUnexpectedTypeData, BDict)
	}
	return n.Dict[key], nil
}

// GetDictStrValue 返回字典键的字符串值；key不存在返回("",nil)；类型错误返回error
func (n *BNode) GetDictStrValue(key string) (string, error) {
	v, err := n.GetDictValue(key)
	if err != nil {
		return "", err
	}
	if v == nil {
		return "", nil // key不存在
	}
	if v.Type != BString {
		return "", fmt.Errorf("%w: expect Bnode Dict value type %s, got %s", ErrUnexpectedTypeData, BString, v.Type)
	}
	getString, err := v.ToString()
	if err != nil {
		return "", err
	}
	return getString, nil
}

// GetDictIntValue 返回字典键的整数值；key不存在返回(0,nil)；类型错误返回error
func (n *BNode) GetDictIntValue(key string) (int, error) {
	v, err := n.GetDictValue(key)
	if err != nil {
		return 0, err
	}
	if v == nil {
		return 0, nil // key不存在
	}
	if v.Type != BInt {
		return 0, fmt.Errorf("%w: expect Bnode Dict value type %s, got %s", ErrUnexpectedTypeData, BInt, v.Type)
	}
	return v.Int, nil
}
