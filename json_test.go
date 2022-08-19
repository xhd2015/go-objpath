package objpath_test

import (
	"encoding/json"
	"testing"
)

// for unmarshal, must reverse the process, e.g.:
//   traverse all fields reversely, and skip seen field

type root struct {
	int
	*int64
	intf
	structA
	*structB
	structC
	structD `json:"-"`

	A0 int    // win
	A1 string // win
	A2 string // win
}

type intf interface{}

type structA struct {
	A0 int    // ignored
	A1 string `json:"sub_a1"` // coexists with root.A1
	A2 string `json:"-"`      // ignored
	A3 string `json:"-"`      // ignored
	A4 string // win only when on structB.A4 not exists, if they both exist: referencing A4 on root shows 'ambiguous selector A4'
	A5 string // win, coexists with structB.A5
}
type structB struct { // ignored
	B0 string
	A4 string //
	A5 string `json:"b_a5"` // win
}
type structC struct {
	C0 string // win
}
type structD struct { // ignored
	D0 string
}

// go test -run TestJSON -v ./src/objpath
func TestJSON(t *testing.T) {

	i := int64(20)
	ro := &root{
		int:   10,
		int64: &i,
		structA: structA{
			A0: 200,
			A1: "201",
			A2: "202",
			A3: "203",
			A4: "204",
			A5: "205",
		},
		structB: &structB{
			B0: "300",
			A4: "304",
			A5: "306",
		},
		structC: structC{
			C0: "400",
		},
		structD: structD{
			D0: "500",
		},

		A0: 100,
		A1: "101",
		A2: "102",
	}
	data, err := json.Marshal(ro)
	if err != nil {
		t.Fatal(err)
	}

	dataJSON := string(data)
	t.Logf("json:%v", dataJSON)
	expectdataJSON := `{"sub_a1":"201","A5":"205","B0":"300","b_a5":"306","C0":"400","A0":100,"A1":"101","A2":"102"}`
	if dataJSON != expectdataJSON {
		t.Fatalf("expect %s = %+v, actual:%+v", `dataJSON`, expectdataJSON, dataJSON)
	}
}
