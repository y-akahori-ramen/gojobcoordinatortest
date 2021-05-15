package gojobcoordinatortest_test

import (
	"reflect"
	"testing"

	"github.com/y-akahori-ramen/gojobcoordinatortest"
)

type sampleStruct struct {
	IntValue    int
	FloatValue  float64
	StringArray []string
}

func TestApiUtil(t *testing.T) {
	testVal := sampleStruct{IntValue: 0, FloatValue: -10, StringArray: []string{"A", "B", "C"}}
	mapData, err := gojobcoordinatortest.StructToMap(testVal)
	if err != nil {
		t.Fatal(err)
	}

	var testVal2 sampleStruct
	err = gojobcoordinatortest.MapToStruct(mapData, &testVal2)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(testVal, testVal2) {
		t.Fail()
	}
}
