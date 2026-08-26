package bencode

import (
	"errors"
	"testing"
)

// ==================== 解码测试 ====================

func TestDecodeInt(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		wantVal int64
		wantPos int
		wantErr bool
		errIs   error
	}{
		{"positive int", []byte("i42e"), 42, 4, false, nil},
		{"negative int", []byte("i-100e"), -100, 6, false, nil},
		{"zero int", []byte("i0e"), 0, 3, false, nil},
		{"invalid -0", []byte("i-0e"), 0, 0, true, ErrInvalidData},
		{"invalid non-numeric", []byte("i12a3e"), 0, 0, true, ErrInvalidData},
		{"truncated no e", []byte("i123"), 0, 0, true, ErrUnexpectedEOF},
		{"overflow int64", []byte("i9223372036854775808e"), 0, 0, true, ErrInvalidData},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := 0
			n, err := DecodeValue(tt.input, &pos)
			if tt.wantErr {
				if err == nil {
					t.Fatal("want error got nil")
				}
				if tt.errIs != nil && !errors.Is(err, tt.errIs) {
					t.Errorf("error should wrap %v, got %v", tt.errIs, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			v, err := n.GetInt()
			if err != nil {
				t.Fatal(err)
			}
			if v != tt.wantVal {
				t.Errorf("val want %d got %d", tt.wantVal, v)
			}
			if pos != tt.wantPos {
				t.Errorf("pos want %d got %d", tt.wantPos, pos)
			}
		})
	}
}

func TestDecodeString(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		wantStr string
		wantPos int
		wantErr bool
		errIs   error
	}{
		{"normal string", []byte("4:spam"), "spam", 6, false, nil},
		{"empty string 0:", []byte("0:"), "", 2, false, nil},
		{"length out of range", []byte("10:abc"), "", 0, true, ErrUnexpectedEOF},
		{"negative length", []byte("-1:abc"), "", 0, true, ErrInvalidData},
		{"non-numeric length", []byte("abc:def"), "", 0, true, ErrInvalidData},
		{"truncated before colon", []byte("123"), "", 0, true, ErrUnexpectedEOF},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := 0
			n, err := DecodeValue(tt.input, &pos)
			if tt.wantErr {
				if err == nil {
					t.Fatal("want error")
				}
				if tt.errIs != nil && !errors.Is(err, tt.errIs) {
					t.Errorf("error should wrap %v, got %v", tt.errIs, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			s, err := n.GetString()
			if err != nil {
				t.Fatal(err)
			}
			if s != tt.wantStr {
				t.Errorf("str want %q got %q", tt.wantStr, s)
			}
			if pos != tt.wantPos {
				t.Errorf("pos want %d got %d", tt.wantPos, pos)
			}
		})
	}
}

func TestDecodeList(t *testing.T) {
	t.Run("normal list", func(t *testing.T) {
		pos := 0
		n, err := DecodeValue([]byte("l4:spami1278ee"), &pos)
		if err != nil {
			t.Fatalf("decode list: %v", err)
		}
		if n.Type != BList || len(n.List) != 2 {
			t.Fatalf("unexpected: %+v", n)
		}
		list, err := n.GetList()
		if err != nil {
			t.Fatal(err)
		}
		getString, err := list[0].GetString()
		if err != nil {
			t.Fatal(err)
		}
		getInt, err := list[1].GetInt()
		if err != nil {
			t.Fatal(err)
		}
		if getString != "spam" || getInt != 1278 {
			t.Fatalf("list items wrong: %+v", n.List)
		}
		if pos != 14 {
			t.Fatalf("unexpected pos: %+v", pos)
		}
	})

	t.Run("empty list le", func(t *testing.T) {
		pos := 0
		n, err := DecodeValue([]byte("le"), &pos)
		if err != nil {
			t.Fatal(err)
		}
		if n.Type != BList || len(n.List) != 0 {
			t.Fatal("empty list fail")
		}
		if pos != 2 {
			t.Errorf("pos want 2 got %d", pos)
		}
	})

	t.Run("nested list", func(t *testing.T) {
		pos := 0
		n, err := DecodeValue([]byte("lli1ei2eee"), &pos)
		if err != nil {
			t.Fatal(err)
		}
		if n.Type != BList || len(n.List) != 1 {
			t.Fatal("nested list outer fail")
		}
		inner := n.List[0]
		if inner.Type != BList || len(inner.List) != 2 {
			t.Fatal("nested list inner fail")
		}
	})

	t.Run("truncated list no e", func(t *testing.T) {
		pos := 0
		_, err := DecodeValue([]byte("li123"), &pos)
		if err == nil {
			t.Fatal("want error for truncated list")
		}
		if !errors.Is(err, ErrUnexpectedEOF) {
			t.Errorf("want ErrUnexpectedEOF, got %v", err)
		}
	})

	t.Run("list element parse error", func(t *testing.T) {
		pos := 0
		_, err := DecodeValue([]byte("lx"), &pos)
		if err == nil {
			t.Fatal("want error for invalid list element")
		}
		if !errors.Is(err, ErrInvalidData) {
			t.Errorf("want ErrInvalidData, got %v", err)
		}
	})
}

func TestDecodeDict(t *testing.T) {
	t.Run("normal dict", func(t *testing.T) {
		pos := 0
		n, err := DecodeValue([]byte("d3:cow3:moo4:spami123ee"), &pos)
		if err != nil {
			t.Fatalf("decode dict: %v", err)
		}
		if n.Type != BDict {
			t.Fatalf("not a dict: %+v", n)
		}
		value, err := n.GetDictStrValue("cow")
		if err != nil {
			t.Fatal(err)
		}
		if value != "moo" {
			t.Fatalf("cow=%q want moo", value)
		}
		v, err := n.GetDictIntValue("spam")
		if err != nil {
			t.Fatal(err)
		}
		if v != 123 {
			t.Fatalf("spam=%d want 123", v)
		}
		if pos != 23 {
			t.Fatalf("unexpected pos: %+v", pos)
		}
	})

	t.Run("empty dict de", func(t *testing.T) {
		pos := 0
		n, err := DecodeValue([]byte("de"), &pos)
		if err != nil {
			t.Fatal(err)
		}
		if n.Type != BDict || len(n.Dict) != 0 {
			t.Fatal("empty dict fail")
		}
		if pos != 2 {
			t.Errorf("pos want 2 got %d", pos)
		}
	})

	t.Run("dict key not exist", func(t *testing.T) {
		pos := 0
		n, err := DecodeValue([]byte("d3:cow3:mooe"), &pos)
		if err != nil {
			t.Fatal(err)
		}
		s, err := n.GetDictStrValue("not_exist")
		if err != nil {
			t.Fatal("key not exist should return nil error")
		}
		if s != "" {
			t.Error("want empty string")
		}

		iv, err := n.GetDictIntValue("not_exist_int")
		if err != nil {
			t.Fatal("key not exist should return nil error")
		}
		if iv != 0 {
			t.Error("want 0")
		}
	})

	t.Run("dict value type mismatch", func(t *testing.T) {
		pos := 0
		n, err := DecodeValue([]byte("d3:key3:vale"), &pos)
		if err != nil {
			t.Fatal(err)
		}
		_, err = n.GetDictIntValue("key")
		if err == nil {
			t.Fatal("want type mismatch error")
		}
		if !errors.Is(err, ErrUnexpectedTypeData) {
			t.Errorf("want ErrUnexpectedTypeData, got %v", err)
		}
	})

	t.Run("truncated dict no e", func(t *testing.T) {
		pos := 0
		_, err := DecodeValue([]byte("d3:keyi123"), &pos)
		if err == nil {
			t.Fatal("want error for truncated dict")
		}
		if !errors.Is(err, ErrUnexpectedEOF) {
			t.Errorf("want ErrUnexpectedEOF, got %v", err)
		}
	})

	t.Run("dict duplicate key overwrite", func(t *testing.T) {
		pos := 0
		n, err := DecodeValue([]byte("d3:keyi1e3:keyi2ee"), &pos)
		if err != nil {
			t.Fatal(err)
		}
		v, err := n.GetDictIntValue("key")
		if err != nil {
			t.Fatal(err)
		}
		if v != 2 {
			t.Errorf("duplicate key should overwrite, want 2 got %d", v)
		}
	})

	t.Run("dict key parse error", func(t *testing.T) {
		pos := 0
		_, err := DecodeValue([]byte("dx"), &pos)
		if err == nil {
			t.Fatal("want error for invalid dict key")
		}
	})
}

func TestDecodeValue_Invalid(t *testing.T) {
	type caseItem struct {
		name  string
		input []byte
	}
	cases := []caseItem{
		{"invalid start byte x", []byte("x")},
		{"truncated int i", []byte("i")},
		{"truncated list l", []byte("l")},
		{"truncated dict d", []byte("d")},
		{"empty data", []byte("")},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			pos := 0
			_, err := DecodeValue(tt.input, &pos)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestDecode_FullParser(t *testing.T) {
	t.Run("full ok", func(t *testing.T) {
		_, err := Decode([]byte("i42e"))
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("full trailing bytes error", func(t *testing.T) {
		_, err := Decode([]byte("i42eXXX"))
		if err == nil {
			t.Fatal("want trailing‑bytes error")
		}
		if !errors.Is(err, ErrInvalidData) {
			t.Errorf("want ErrInvalidData, got %v", err)
		}
	})
	t.Run("full parse error propagation", func(t *testing.T) {
		_, err := Decode([]byte("x"))
		if err == nil {
			t.Fatal("want parse error")
		}
	})
}

func TestDecodeValue_EOF(t *testing.T) {
	pos := 3
	data := []byte("abc")
	_, err := DecodeValue(data, &pos)
	if err == nil {
		t.Fatal("want EOF error")
	}
	if !errors.Is(err, ErrUnexpectedEOF) {
		t.Errorf("want ErrUnexpectedEOF, got %v", err)
	}
}

// ==================== Getter 边界测试 ====================

func TestGetterNilNode(t *testing.T) {
	var n *BNode = nil
	_, err := n.GetInt()
	if err == nil {
		t.Error("GetInt nil node want error")
	}
	_, err = n.GetString()
	if err == nil {
		t.Error("GetString nil node want error")
	}
	_, err = n.GetHexString()
	if err == nil {
		t.Error("GetHexString nil node want error")
	}
	_, err = n.GetList()
	if err == nil {
		t.Error("GetList nil node want error")
	}
	_, err = n.GetDictValue("x")
	if err == nil {
		t.Error("GetDictValue nil node want error")
	}
	_, err = n.GetDictStrValue("x")
	if err == nil {
		t.Error("GetDictStrValue nil node want error")
	}
	_, err = n.GetDictIntValue("x")
	if err == nil {
		t.Error("GetDictIntValue nil node want error")
	}
}

func TestGetterTypeMismatch(t *testing.T) {
	nodeInt := &BNode{Type: BInt, Int: 10}
	_, err := nodeInt.GetString()
	if err == nil {
		t.Error("int node call GetString want error")
	}
	_, err = nodeInt.GetHexString()
	if err == nil {
		t.Error("int node call GetHexString want error")
	}
	_, err = nodeInt.GetList()
	if err == nil {
		t.Error("int node call GetList want error")
	}
	_, err = nodeInt.GetDictValue("k")
	if err == nil {
		t.Error("int node call GetDictValue want error")
	}

	nodeStr := &BNode{Type: BString, Str: []byte("abc")}
	_, err = nodeStr.GetInt()
	if err == nil {
		t.Error("string node call GetInt want error")
	}
	_, err = nodeStr.GetList()
	if err == nil {
		t.Error("string node call GetList want error")
	}

	nodeList := &BNode{Type: BList}
	_, err = nodeList.GetInt()
	if err == nil {
		t.Error("list node call GetInt want error")
	}
	_, err = nodeList.GetString()
	if err == nil {
		t.Error("list node call GetString want error")
	}
	_, err = nodeList.GetDictValue("k")
	if err == nil {
		t.Error("list node call GetDictValue want error")
	}
}

func TestGetHexString_Normal(t *testing.T) {
	n := &BNode{Type: BString, Str: []byte{0x12, 0x34, 0xab, 0xcd}}
	s, err := n.GetHexString()
	if err != nil {
		t.Fatal(err)
	}
	if s != "1234abcd" {
		t.Errorf("hex want 1234abcd got %q", s)
	}
}

// ==================== 编码测试 ====================

func TestEncodeIndependent(t *testing.T) {
	t.Run("EncodeInt", func(t *testing.T) {
		got := string(EncodeInt(42))
		want := "i42e"
		if got != want {
			t.Errorf("want %q got %q", want, got)
		}
		got = string(EncodeInt(-123))
		want = "i-123e"
		if got != want {
			t.Errorf("want %q got %q", want, got)
		}
	})

	t.Run("EncodeString", func(t *testing.T) {
		got := string(EncodeString("spam"))
		want := "4:spam"
		if got != want {
			t.Errorf("want %q got %q", want, got)
		}
		got = string(EncodeString(""))
		want = "0:"
		if got != want {
			t.Errorf("empty string want %q got %q", want, got)
		}
	})

	t.Run("EncodeBytes", func(t *testing.T) {
		got := string(EncodeBytes([]byte("spam")))
		want := "4:spam"
		if got != want {
			t.Errorf("want %q got %q", want, got)
		}
	})

	t.Run("EncodeList", func(t *testing.T) {
		list := []*BNode{
			{Type: BString, Str: []byte("spam")},
			{Type: BInt, Int: 123},
		}
		got := string(EncodeList(list))
		want := "l4:spami123ee"
		if got != want {
			t.Errorf("want %q got %q", want, got)
		}
		got = string(EncodeList(nil))
		want = "le"
		if got != want {
			t.Errorf("empty list want %q got %q", want, got)
		}
	})

	t.Run("EncodeDict", func(t *testing.T) {
		dict := map[string]*BNode{
			"zebra": {Type: BString, Str: []byte("x")},
			"apple": {Type: BString, Str: []byte("y")},
		}
		got := string(EncodeDict(dict))
		want := "d5:apple1:y5:zebra1:xe"
		if got != want {
			t.Errorf("want %q got %q", want, got)
		}
		got = string(EncodeDict(nil))
		want = "de"
		if got != want {
			t.Errorf("empty dict want %q got %q", want, got)
		}
	})
}

func TestEncodeRawReuse(t *testing.T) {
	t.Run("int raw reuse", func(t *testing.T) {
		n := &BNode{
			Type: BInt,
			Int:  999,
			Raw:  []byte("i42e"),
		}
		got := string(n.Encode())
		want := "i42e"
		if got != want {
			t.Errorf("raw reuse failed, want %q got %q", want, got)
		}
	})

	t.Run("string raw reuse", func(t *testing.T) {
		n := &BNode{
			Type: BString,
			Str:  []byte("wrong"),
			Raw:  []byte("4:spam"),
		}
		got := string(n.Encode())
		want := "4:spam"
		if got != want {
			t.Errorf("raw reuse failed, want %q got %q", want, got)
		}
	})

	t.Run("parsed node auto raw", func(t *testing.T) {
		original := "i42e"
		pos := 0
		n, err := DecodeValue([]byte(original), &pos)
		if err != nil {
			t.Fatal(err)
		}
		if len(n.Raw) == 0 {
			t.Fatal("parsed node should have Raw field")
		}
		got := string(n.Encode())
		if got != original {
			t.Errorf("round trip want %q got %q", original, got)
		}
	})
}

func TestEncodeRoundTrip(t *testing.T) {
	cases := []string{
		"i42e",
		"i-123e",
		"4:spam",
		"0:",
		"le",
		"de",
		"l4:spam4:eggse",
		"d3:cow3:moo4:spam4:eggse",
	}
	for _, tc := range cases {
		t.Run(tc, func(t *testing.T) {
			pos := 0
			n, err := DecodeValue([]byte(tc), &pos)
			if err != nil {
				t.Fatalf("decode %q: %v", tc, err)
			}
			got := string(n.Encode())
			if got != tc {
				t.Errorf("round trip %q -> %q", tc, got)
			}
			if pos != len(tc) {
				t.Errorf("pos mismatch %q pos=%d", tc, pos)
			}
		})
	}
}

func TestEncodeDictSortedKeys(t *testing.T) {
	n := &BNode{
		Type: BDict,
		Dict: map[string]*BNode{
			"zebra": {Type: BString, Str: []byte("x")},
			"apple": {Type: BString, Str: []byte("y")},
		},
	}
	got := string(n.Encode())
	want := "d5:apple1:y5:zebra1:xe"
	if got != want {
		t.Errorf("sorted dict encode = %q, want %q", got, want)
	}
}

func TestEncodeUnknownType(t *testing.T) {
	n := &BNode{Type: BType(99)}
	got := n.Encode()
	if len(got) != 0 {
		t.Errorf("unknown type encode should return empty, got len %d", len(got))
	}
}

// ==================== 类型定义测试 ====================

func TestBTypeString(t *testing.T) {
	tests := []struct {
		bt   BType
		want string
	}{
		{BInt, "BInt"},
		{BString, "BString"},
		{BList, "BList"},
		{BDict, "BDict"},
		{BType(99), "unknown(99)"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.bt.String()
			if got != tt.want {
				t.Errorf("want %q got %q", tt.want, got)
			}
		})
	}
}

// ==================== 复杂嵌套测试 ====================

func TestNestedStructure(t *testing.T) {
	peers := &BNode{Type: BList, List: []*BNode{
		{Type: BList, List: []*BNode{{Type: BString, Str: []byte("1.2.3.4")}, {Type: BInt, Int: 6881}}},
		{Type: BList, List: []*BNode{{Type: BString, Str: []byte("5.6.7.8")}, {Type: BInt, Int: 9999}}},
	}}
	n := &BNode{Type: BDict, Dict: map[string]*BNode{
		"complete":   {Type: BInt, Int: 5},
		"incomplete": {Type: BInt, Int: 3},
		"peers":      peers,
	}}
	data := n.Encode()
	pos := 0
	got, err := DecodeValue(data, &pos)
	if err != nil {
		t.Fatalf("decode nested: %v", err)
	}

	if v, err := got.GetDictIntValue("complete"); v != 5 || err != nil {
		t.Errorf("complete=%d want 5", v)
	}
	if v, err := got.GetDictIntValue("incomplete"); v != 3 || err != nil {
		t.Errorf("incomplete=%d want 3", v)
	}

	gotPeers, err := got.GetDictValue("peers")
	if err != nil || gotPeers == nil || gotPeers.Type != BList || len(gotPeers.List) != 2 {
		t.Fatalf("peers wrong: %+v", gotPeers)
	}

	// 逐个校验 peer0
	peer0 := gotPeers.List[0]
	if peer0.Type != BList || len(peer0.List) != 2 {
		t.Fatal("peer0 list wrong")
	}
	p0ip, _ := peer0.List[0].GetString()
	p0port, _ := peer0.List[1].GetInt()
	if p0ip != "1.2.3.4" || p0port != 6881 {
		t.Errorf("peer0 want 1.2.3.4:6881, got %s:%d", p0ip, p0port)
	}

	// 逐个校验 peer1
	peer1 := gotPeers.List[1]
	if peer1.Type != BList || len(peer1.List) != 2 {
		t.Fatal("peer1 list wrong")
	}
	p1ip, _ := peer1.List[0].GetString()
	p1port, _ := peer1.List[1].GetInt()
	if p1ip != "5.6.7.8" || p1port != 9999 {
		t.Errorf("peer1 want 5.6.7.8:9999, got %s:%d", p1ip, p1port)
	}
}

func TestDeepNested(t *testing.T) {
	input := []byte("llllli42eeeeee") // 5层list + 1个整数e + 5个list闭合e，共6个e
	pos := 0
	n, err := DecodeValue(input, &pos)
	if err != nil {
		t.Fatal(err)
	}
	cur := n
	for i := 0; i < 4; i++ {
		if len(cur.List) != 1 {
			t.Fatalf("level %d list len != 1", i)
		}
		cur = cur.List[0]
		if cur.Type != BList {
			t.Fatalf("level %d not list", i+1)
		}
	}
	if len(cur.List) != 1 || cur.List[0].Type != BInt {
		t.Fatal("inner most not int")
	}
	v, _ := cur.List[0].GetInt()
	if v != 42 {
		t.Errorf("inner int want 42 got %d", v)
	}
}

// ==================== 模糊测试 ====================

// FuzzDecode 纯解码模糊测试：任意字节输入下不崩溃、不越界
func FuzzDecode(f *testing.F) {
	// 种子语料：覆盖合法、非法、截断、空、嵌套等典型场景
	seeds := [][]byte{
		[]byte("i42e"),
		[]byte("i-123e"),
		[]byte("i0e"),
		[]byte("4:spam"),
		[]byte("0:"),
		[]byte("le"),
		[]byte("de"),
		[]byte("l4:spami123ee"),
		[]byte("d3:key3:valuee"),
		[]byte("llllli42eeeeee"),
		[]byte(""),
		[]byte("i"),
		[]byte("l"),
		[]byte("d"),
		[]byte("x"),
		[]byte("i123"),
		[]byte("10:abc"),
		[]byte("d3:keyi123"),
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// 顶层完整解码
		_, _ = Decode(data)

		// 指针模式解码
		pos := 0
		_, _ = DecodeValue(data, &pos)
	})
}

// FuzzEncodeDecodeRoundTrip 编解码往返模糊测试
// 对解码成功的数据，编码后再解码，验证语义一致性
func FuzzEncodeDecodeRoundTrip(f *testing.F) {
	seeds := [][]byte{
		[]byte("i42e"),
		[]byte("4:spam"),
		[]byte("le"),
		[]byte("de"),
		[]byte("l4:spam4:eggse"),
		[]byte("d3:cow3:moo4:spam4:eggse"),
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		node1, err := Decode(data)
		if err != nil {
			return // 非法输入直接跳过，只测合法输入的往返一致性
		}

		encoded := node1.Encode()
		node2, err := Decode(encoded)
		if err != nil {
			t.Fatalf("编码输出无法解码: %v", err)
		}

		if !nodeEqual(node1, node2) {
			t.Fatalf("往返不一致\n原始: %+v\n编码: %s\n再解码: %+v", node1, string(encoded), node2)
		}
	})
}

// nodeEqual 深度比较两个 BNode 语义是否相等，忽略 Raw 字段
func nodeEqual(a, b *BNode) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Type != b.Type {
		return false
	}

	switch a.Type {
	case BInt:
		return a.Int == b.Int
	case BString:
		return string(a.Str) == string(b.Str)
	case BList:
		if len(a.List) != len(b.List) {
			return false
		}
		for i := range a.List {
			if !nodeEqual(a.List[i], b.List[i]) {
				return false
			}
		}
		return true
	case BDict:
		if len(a.Dict) != len(b.Dict) {
			return false
		}
		for k, va := range a.Dict {
			vb, ok := b.Dict[k]
			if !ok {
				return false
			}
			if !nodeEqual(va, vb) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// ==================== 基准测试 ====================

// ---------- 解码性能 ----------

func BenchmarkDecodeInt(b *testing.B) {
	data := []byte("i9223372036854775807e")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pos := 0
		_, _ = DecodeValue(data, &pos)
	}
}

func BenchmarkDecodeString(b *testing.B) {
	data := []byte("32:abcdefghijklmnopqrstuvwxyz012345")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pos := 0
		_, _ = DecodeValue(data, &pos)
	}
}

func BenchmarkDecodeList(b *testing.B) {
	data := []byte("li1ei2ei3ei4ei5ei6ei7ei8ei9ei10ee")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pos := 0
		_, _ = DecodeValue(data, &pos)
	}
}

func BenchmarkDecodeDict(b *testing.B) {
	data := []byte("d1:a1:11:b1:21:c1:31:d1:41:e1:5e")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pos := 0
		_, _ = DecodeValue(data, &pos)
	}
}

// ---------- 编码性能 ----------

func BenchmarkEncodeInt(b *testing.B) {
	node := &BNode{Type: BInt, Int: 9223372036854775807}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = node.Encode()
	}
}

func BenchmarkEncodeString(b *testing.B) {
	node := &BNode{Type: BString, Str: []byte("abcdefghijklmnopqrstuvwxyz012345")}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = node.Encode()
	}
}

func BenchmarkEncodeList(b *testing.B) {
	list := make([]*BNode, 10)
	for i := 0; i < 10; i++ {
		list[i] = &BNode{Type: BInt, Int: int64(i)}
	}
	node := &BNode{Type: BList, List: list}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = node.Encode()
	}
}

func BenchmarkEncodeDict(b *testing.B) {
	dict := map[string]*BNode{
		"a": {Type: BInt, Int: 1},
		"b": {Type: BInt, Int: 2},
		"c": {Type: BInt, Int: 3},
		"d": {Type: BInt, Int: 4},
		"e": {Type: BInt, Int: 5},
	}
	node := &BNode{Type: BDict, Dict: dict}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = node.Encode()
	}
}

// BenchmarkEncodeWithRaw 测试 Raw 字节复用的编码性能（解析生成的节点）
func BenchmarkEncodeWithRaw(b *testing.B) {
	data := []byte("d3:key3:valuee")
	node, _ := Decode(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = node.Encode()
	}
}

// BenchmarkEncodeWithoutRaw 测试无 Raw 字段重新编码的性能（手动构造的节点）
func BenchmarkEncodeWithoutRaw(b *testing.B) {
	dict := map[string]*BNode{
		"key": {Type: BString, Str: []byte("value")},
	}
	node := &BNode{Type: BDict, Dict: dict}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = node.Encode()
	}
}
