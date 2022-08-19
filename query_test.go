package objpath

import (
	"fmt"
	"testing"
)

// go test -run TestQuerySimple -v ./src/objpath
func TestQuerySimple(t *testing.T) {
	v, err := Query(map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": 23345,
			},
		},
	}, "a.b.c")
	if err != nil {
		t.Fatal(err)
	}

	s := fmt.Sprintf("%v", v)
	expect := `[23345]`
	if s != expect {
		t.Fatalf("expect %s = %+v, actual:%+v", `s`, expect, s)
	}
}
