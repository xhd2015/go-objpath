package objpath

import (
	"testing"
)

type Options string

const (
	OptionFail Options = "fail"
)

func testAssert(t *testing.T, obj interface{}, filter string, assert string, opts ...Options) {
	jsonFilter, err := parseJSONFilter(filter)
	if err != nil {
		t.Fatal(err)
	}
	val := NewObject(obj)
	liveVals, res := jsonFilter.Filter([]Object{val}, val)
	// t.Logf("live: %v", liveVals)
	needPass := true
	for _, opt := range opts {
		if opt == OptionFail {
			needPass = false
			break
		}
	}
	if needPass {
		AssertOkT(t, "res ok", res.Ok())
	} else {
		AssertNotOkT(t, "res ok", !res.Ok())
	}
	AssertT(t, liveVals, assert)
}

// go test -run TestFilterSimpleMap -v ./
func TestFilterSimpleMap(t *testing.T) {
	testAssert(t,
		map[string]interface{}{
			"a": map[string]interface{}{
				"b": map[string]interface{}{
					"c": 23345,
				},
			},
		},
		`{
			"a.b.c":23345
		}`,
		`{"0.Value.a.b.c":"23345"}`,
	)
}

// go test -run TestFilterMultiConditions -v ./
func TestFilterMultiConditions(t *testing.T) {
	testAssert(t,
		map[string]interface{}{
			"a": map[string]interface{}{
				"b": map[string]interface{}{
					"c": 23345,
				},
				"d": "123",
			},
		},
		`{
			"a.b.c":"23345",
			"a.d":{
				"$gt":"100"
			}
		}`,
		`{
			"0.Value.a.b.c":"23345",
			"0.Value.a.d":123
		}`,
	)
}

// go test -run TestFilterNonMatch -v ./
func TestFilterNonMatch(t *testing.T) {
	testAssert(t,
		map[string]interface{}{
			"a": map[string]interface{}{
				"b": map[string]interface{}{
					"c": 23345,
				},
				"d": "1_2BB_3",
			},
		},
		`{
			"a.b.c":"23345",
			"a.d":{
				"$contains":"2BB_4"
			}
		}`,
		`{
			"$length":"0"
		}`,
		OptionFail,
	)
}

// go test -run TestFilterWildcardMatch -v ./
func TestFilterWildcardMatch(t *testing.T) {
	testAssert(t,
		map[string]interface{}{
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
		},
		`{
			"a.b.*":"23345",
			"a.d.*":{
				"$contains":"234"
			}
		}`,
		`{
			"$length":"1"
		}`,
	)
}

// go test -run TestFilterWildcardNotAny -v ./
func TestFilterWildcardNotAny(t *testing.T) {
	testAssert(t,
		map[string]interface{}{
			"a": map[string]interface{}{
				"b": map[string]interface{}{
					"c": 23345,
				},
				"d": map[string]interface{}{
					"e": "234",
					"f": "1234",
				},
			},
		},
		`{
			"a.b.*":"23345",
			"a.d.*":{
				"$contains":"1x234"
			}
		}`,
		`{
			"$length":"0"
		}`,
		OptionFail,
	)
}

// go test -run TestFilterNestedNonMatch -v ./
func TestFilterNestedNonMatch(t *testing.T) {
	testAssert(t,
		map[string]interface{}{
			"a": map[string]interface{}{
				"b": map[string]interface{}{
					"c": 23345,
				},
				"d": map[string]interface{}{
					"e": "234",
					"f": "1234",
				},
			},
		},
		`{
			"a":{
				"b":{
					"c":"23345"
				},
				"d":{
					"e":"2345"
				}
			}
		}`,
		`{
			"$length":"0"
		}`,
		OptionFail,
	)
}

// go test -run TestFilterNestedHasMatch -v ./
func TestFilterNestedHasMatch(t *testing.T) {
	testAssert(t,
		map[string]interface{}{
			"a": map[string]interface{}{
				"b": map[string]interface{}{
					"c": 23345,
				},
				"d": map[string]interface{}{
					"e": "234",
					"f": "1234",
				},
			},
		},
		`{
			"a":{
				"b":{
					"c":"23345"
				},
				"d":{
					"e":"234",
					"f":"1234"
				}
			}
		}`,
		`{
			"$length":"1"
		}`,
	)
}

// go test -run TestFilterRefOtherPart -v ./
func TestFilterRefOtherPart(t *testing.T) {
	testAssert(t,
		map[string]interface{}{
			"a": map[string]interface{}{
				"b": map[string]interface{}{
					"c": 23345,
				},
				"d": map[string]interface{}{
					"e": "234",
					"f": "1234",
				},
			},
		},
		`{
			"a.b.c":{
				"$gt":"$.a.d.*"
			}
		}`,
		`{
			"$length":"1"
		}`,
	)
}

// go test -run TestFilterFlattenSimple -v ./
func TestFilterFlattenSimple(t *testing.T) {
	testAssert(t,
		map[string]interface{}{
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
		},
		`{
			"#a.b.[c.f]":"=> ok",
			"#a.b[c.f]":"ok",
			"a.b.[c.d]":{
				"$endsWith":"45"
			}
		}`,
		`{
			"$length":"1"
		}`,
	)
}

// go test -run TestFilterFlattenWildcardEactlyOneMatch -v ./
func TestFilterFlattenWildcardEactlyOneMatch(t *testing.T) {
	// for wildcard, only need one match means hit
	testAssert(t,
		map[string]interface{}{
			"a": map[string]interface{}{
				"b": map[string]interface{}{
					// "c.dx": 233456,
					"c.fx": "445",
				},
				"d": map[string]interface{}{
					"e": "234",
					"f": "1234",
					// "c.x": "1456",
				},
			},
		},
		`{
			"a.*.[c.*x]":{
				"$endsWith":"45"
			}
		}`,
		`{
			"$length":"1"
		}`,
	)
}

// go test -run TestFilterFlattenWildcardOnlyOneForMultiFields -v ./
func TestFilterFlattenWildcardOnlyOneForMultiFields(t *testing.T) {
	// for wildcard, only need one match means hit
	testAssert(t,
		map[string]interface{}{
			"a": map[string]interface{}{
				"b": map[string]interface{}{
					"c.dx": 233456,
					"c.fx": "445",
				},
				"d": map[string]interface{}{
					"e":   "234",
					"f":   "1234",
					"c.x": "1456",
				},
			},
		},
		`{
			"a.*.[c.*x]":{
				"$endsWith":"45"
			}
		}`,
		`{
			"$length":"1"
		}`,
	)
}
