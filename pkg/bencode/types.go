package bencode

import (
	"fmt"
)

type BType int

const (
	BInt BType = iota
	BString
	BList
	BDict
)

func (t BType) String() string {
	switch t {
	case BInt:
		return "BInt"
	case BString:
		return "BString"
	case BList:
		return "BList"
	case BDict:
		return "BDict"
	default:
		return fmt.Sprintf("unknown(%d)", t)
	}
}

type BNode struct {
	Type     BType
	Int      int64
	Str      []byte
	List     []*BNode
	Dict     map[string]*BNode
	DictKeys []string //保存解析出的key，用于编码排序
	Raw      []byte
}
