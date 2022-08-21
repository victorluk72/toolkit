package toolkit

import "testing"

func TestToolsRandomStringGenerator(t *testing.T) {

	var testTools Tools

	s := testTools.RandomStringGenerator(10)

	//Let's test if the length of my string is 10 charachters
	if len(s) != 11 {
		t.Error("wrong length of the variable")
	}

}
