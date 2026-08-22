package bencode

type BType int

const (
	BInt BType = iota
	BString
	BList
	BDict
)

type BNode struct {
	Type BType
	Int  int64
	Str  string
	List []*BNode
	Dict map[string]*BNode
}
