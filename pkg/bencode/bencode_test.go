package bencode

import (
	"reflect"
	"testing"
)

func TestDecodeInt(t *testing.T) {
	n, err := Decode([]byte("i42e"))
	if err != nil {
		t.Fatalf("decode int: %v", err)
	}
	if n.Type != BInt || n.Int != 42 {
		t.Fatalf("unexpected: %+v", n)
	}
}

func TestDecodeString(t *testing.T) {
	n, err := Decode([]byte("4:spam"))
	if err != nil {
		t.Fatalf("decode string: %v", err)
	}
	if n.Type != BString || n.Str != "spam" {
		t.Fatalf("unexpected: %+v", n)
	}
}

func TestDecodeList(t *testing.T) {
	n, err := Decode([]byte("l4:spam4:eggse"))
	if err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if n.Type != BList || len(n.List) != 2 {
		t.Fatalf("unexpected: %+v", n)
	}
	if n.List[0].Str != "spam" || n.List[1].Str != "eggs" {
		t.Fatalf("list items wrong: %+v", n.List)
	}
}

func TestDecodeDict(t *testing.T) {
	n, err := Decode([]byte("d3:cow3:moo4:spam4:eggse"))
	if err != nil {
		t.Fatalf("decode dict: %v", err)
	}
	if n.Type != BDict {
		t.Fatalf("not a dict: %+v", n)
	}
	if got := n.GetString("cow"); got != "moo" {
		t.Fatalf("cow=%q want moo", got)
	}
	if got := n.GetString("spam"); got != "eggs" {
		t.Fatalf("spam=%q want eggs", got)
	}
}

func TestDecodeTrailingBytes(t *testing.T) {
	if _, err := Decode([]byte("i1ee")); err == nil {
		t.Fatal("expected error for trailing bytes")
	}
}

func TestDecodeInvalid(t *testing.T) {
	if _, err := Decode([]byte("x")); err == nil {
		t.Fatal("expected error for invalid byte")
	}
	if _, err := Decode([]byte("i")); err == nil {
		t.Fatal("expected error for truncated int")
	}
}

func TestEncodeRoundTrip(t *testing.T) {
	cases := []string{
		"i42e",
		"4:spam",
		"l4:spam4:eggse",
		"d3:cow3:moo4:spam4:eggse",
	}
	for _, tc := range cases {
		n, err := Decode([]byte(tc))
		if err != nil {
			t.Fatalf("decode %q: %v", tc, err)
		}
		got := string(Encode(n))
		if got != tc {
			t.Errorf("round trip %q -> %q", tc, got)
		}
	}
}

func TestEncodeDictSortedKeys(t *testing.T) {
	// 字典键必须按字典序排序，保证编码结果确定。
	n := &BNode{
		Type: BDict,
		Dict: map[string]*BNode{
			"zebra": {Type: BString, Str: "x"},
			"apple": {Type: BString, Str: "y"},
		},
	}
	got := string(Encode(n))
	want := "d5:apple1:y5:zebra1:xe"
	if got != want {
		t.Errorf("sorted dict encode = %q, want %q", got, want)
	}
}

func TestNestedStructure(t *testing.T) {
	// 模拟真实 announce 场景：字典内嵌列表，列表元素为 [ip字符串, port整数] 子列表。
	// 用编码器生成数据，同时验证编解码对称性。
	peers := &BNode{Type: BList, List: []*BNode{
		{Type: BList, List: []*BNode{{Type: BString, Str: "1.2.3.4"}, {Type: BInt, Int: 6881}}},
		{Type: BList, List: []*BNode{{Type: BString, Str: "5.6.7.8"}, {Type: BInt, Int: 9999}}},
	}}
	n := &BNode{Type: BDict, Dict: map[string]*BNode{
		"complete":   {Type: BInt, Int: 5},
		"incomplete": {Type: BInt, Int: 3},
		"peers":      peers,
	}}
	data := Encode(n)

	got, err := Decode(data)
	if err != nil {
		t.Fatalf("decode nested: %v", err)
	}
	if v := got.GetInt("complete"); v != 5 {
		t.Errorf("complete=%d want 5", v)
	}
	if v := got.GetInt("incomplete"); v != 3 {
		t.Errorf("incomplete=%d want 3", v)
	}
	gotPeers := got.Get("peers")
	if gotPeers == nil || gotPeers.Type != BList || len(gotPeers.List) != 2 {
		t.Fatalf("peers wrong: %+v", gotPeers)
	}
	wantPeer0 := []*BNode{{Type: BString, Str: "1.2.3.4"}, {Type: BInt, Int: 6881}}
	wantPeer1 := []*BNode{{Type: BString, Str: "5.6.7.8"}, {Type: BInt, Int: 9999}}
	if !reflect.DeepEqual(gotPeers.List[0].List, wantPeer0) {
		t.Errorf("peer0 wrong: %+v", gotPeers.List[0].List)
	}
	if !reflect.DeepEqual(gotPeers.List[1].List, wantPeer1) {
		t.Errorf("peer1 wrong: %+v", gotPeers.List[1].List)
	}
}
