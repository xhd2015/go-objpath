package objpath

import (
	"fmt"
	"testing"
)

// go test -run TestFilterSimpleMap -v ./src/objpath
func TestFilterSimpleMap(t *testing.T) {
	filter, err := parseJSONFilter(`{
		"a.b.c":23345
	}`)
	if err != nil {
		t.Fatal(err)
	}

	val := NewObject(map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": 23345,
			},
		},
	})

	liveVals, err := filter.Filter([]Object{val}, val)
	if err != nil {
		t.Fatal(err)
	}
	valsStr := fmt.Sprint(liveVals)
	expect := `[map[a:map[b:map[c:23345]]]]`
	if valsStr != expect {
		t.Fatalf("expect %s = %+v, actual:%+v", `valsStr`, expect, valsStr)
	}
}

// go test -run TestFilterMultiConditions -v ./src/objpath
func TestFilterMultiConditions(t *testing.T) {
	filter, err := parseJSONFilter(`{
		"a.b.c":"23345",
		"a.d":{
			"$gt":"100"
		}
	}`)
	if err != nil {
		t.Fatal(err)
	}

	val := NewObject(map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": 23345,
			},
			"d": "123",
		},
	})

	liveVals, err := filter.Filter([]Object{val}, val)
	if err != nil {
		t.Fatal(err)
	}
	valsStr := fmt.Sprint(liveVals)
	expect := `[map[a:map[b:map[c:23345] d:123]]]`
	if valsStr != expect {
		t.Fatalf("expect %s = %+v, actual:%+v", `valsStr`, expect, valsStr)
	}
}

// go test -run TestFilterNonMatch -v ./src/objpath
func TestFilterNonMatch(t *testing.T) {
	filter, err := parseJSONFilter(`{
		"a.b.c":"23345",
		"a.d":{
			"$contains":"2BB_4"
		}
	}`)
	if err != nil {
		t.Fatal(err)
	}

	val := NewObject(map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": 23345,
			},
			"d": "1_2BB_3",
		},
	})

	liveVals, err := filter.Filter([]Object{val}, val)
	if err != nil {
		t.Fatal(err)
	}
	valsStr := fmt.Sprint(liveVals)
	expect := `[]`
	if valsStr != expect {
		t.Fatalf("expect %s = %+v, actual:%+v", `valsStr`, expect, valsStr)
	}
}

// go test -run TestFilterWildcard -v ./src/objpath
func TestFilterWildcard(t *testing.T) {
	filter, err := parseJSONFilter(`{
		"a.b.*":"23345",
		"a.d.*":{
			"$contains":"234"
		}
	}`)
	if err != nil {
		t.Fatal(err)
	}

	val := NewObject(map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": 23345,
			},
			"d": map[string]interface{}{
				// "123",
				"e": "234",
				"f": "1234",
			},
		},
	})

	liveVals, err := filter.Filter([]Object{val}, val)
	if err != nil {
		t.Fatal(err)
	}
	valsStr := len(liveVals)
	expect := 1
	if valsStr != expect {
		t.Fatalf("expect %s = %+v, actual:%+v", `valsStr`, expect, valsStr)
	}
}

// go test -run TestFilterWildcardNotAny -v ./src/objpath
func TestFilterWildcardNotAny(t *testing.T) {
	filter, err := parseJSONFilter(`{
		"a.b.*":"23345",
		"a.d.*":{
			"$contains":"1x234"
		}
	}`)
	if err != nil {
		t.Fatal(err)
	}

	val := NewObject(map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": 23345,
			},
			"d": map[string]interface{}{
				"e": "234",
				"f": "1234",
			},
		},
	})

	liveVals, err := filter.Filter([]Object{val}, val)
	if err != nil {
		t.Fatal(err)
	}
	valsStr := len(liveVals)
	expect := 0
	if valsStr != expect {
		t.Fatalf("expect %s = %+v, actual:%+v", `valsStr`, expect, valsStr)
	}
}

// go test -run TestFilterNestedNonMatch -v ./src/objpath
func TestFilterNestedNonMatch(t *testing.T) {
	filter, err := parseJSONFilter(`{
		"a":{
			"b":{
				"c":"23345"
			},
			"d":{
				"e":"2345"
			}
		}
	}`)
	if err != nil {
		t.Fatal(err)
	}

	val := NewObject(map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": 23345,
			},
			"d": map[string]interface{}{
				"e": "234",
				"f": "1234",
			},
		},
	})

	liveVals, err := filter.Filter([]Object{val}, val)
	if err != nil {
		t.Fatal(err)
	}
	valsStr := len(liveVals)
	expect := 0
	if valsStr != expect {
		t.Fatalf("expect %s = %+v, actual:%+v", `valsStr`, expect, valsStr)
	}
}

// go test -run TestFilterNestedHasMatch -v ./src/objpath
func TestFilterNestedHasMatch(t *testing.T) {
	filter, err := parseJSONFilter(`{
		"a":{
			"b":{
				"c":"23345"
			},
			"d":{
				"e":"234",
				"f":"1234"
			}
		}
	}`)
	if err != nil {
		t.Fatal(err)
	}

	val := NewObject(map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": 23345,
			},
			"d": map[string]interface{}{
				"e": "234",
				"f": "1234",
			},
		},
	})

	liveVals, err := filter.Filter([]Object{val}, val)
	if err != nil {
		t.Fatal(err)
	}
	valsStr := len(liveVals)
	expect := 1
	if valsStr != expect {
		t.Fatalf("expect %s = %+v, actual:%+v", `valsStr`, expect, valsStr)
	}
}

// go test -run TestFilterRefOtherPart -v ./src/objpath
func TestFilterRefOtherPart(t *testing.T) {
	filter, err := parseJSONFilter(`{
		"a.b.c":{
			"$gt":"$.a.d.*"
		}
	}`)
	if err != nil {
		t.Fatal(err)
	}

	val := NewObject(map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": 23345,
			},
			"d": map[string]interface{}{
				"e": "234",
				"f": "1234",
			},
		},
	})

	liveVals, err := filter.Filter([]Object{val}, val)
	if err != nil {
		t.Fatal(err)
	}
	valsStr := len(liveVals)
	expect := 1
	if valsStr != expect {
		t.Fatalf("expect %s = %+v, actual:%+v", `valsStr`, expect, valsStr)
	}
}

// go test -run TestFilterFlattenSimple -v ./src/objpath
func TestFilterFlattenSimple(t *testing.T) {
	filter, err := parseJSONFilter(`{
		"#a.b.[c.f]":"=> ok",
		"#a.b[c.f]":"ok",
		"a.b.[c.d]":{
			"$endsWith":"45"
		}
	}`)
	if err != nil {
		t.Fatal(err)
	}

	val := NewObject(map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c.d": 23345,
				"c.f": "445",
			},
			"d": map[string]interface{}{
				"e": "234",
				"f": "1234",
			},
		},
	})

	liveVals, err := filter.Filter([]Object{val}, val)
	if err != nil {
		t.Fatal(err)
	}
	valsStr := len(liveVals)
	expect := 1
	if valsStr != expect {
		t.Fatalf("expect %s = %+v, actual:%+v", `valsStr`, expect, valsStr)
	}
}

// NOTE: currently only support suffix or prefix glob
// go test -run TestFilterFlattenWildcard -v ./src/objpath
func TestFilterFlattenWildcard(t *testing.T) {
	filter, err := parseJSONFilter(`{
		"a.b.[c.*]":{
			"$endsWith":"45"
		}
	}`)
	if err != nil {
		t.Fatal(err)
	}

	val := NewObject(map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c.d": 23345,
				"c.f": "4456",
			},
			"d": map[string]interface{}{
				"e": "234",
				"f": "1234",
			},
		},
	})

	liveVals, err := filter.Filter([]Object{val}, val)
	if err != nil {
		t.Fatal(err)
	}
	valsStr := len(liveVals)
	expect := 1
	if valsStr != expect {
		t.Fatalf("expect %s = %+v, actual:%+v", `valsStr`, expect, valsStr)
	}
}
