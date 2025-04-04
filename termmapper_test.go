package termmapper

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConstants(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(Hui(), "test")
}

// KoralPipe-TermMapping

func TestTokenBuilder(t *testing.T) {

	assert := assert.New(t)

	var strBuilder strings.Builder
	term(&strBuilder, "myfoundry", "mylayer", "mykey1", true)
	assert.Equal(strBuilder.String(), `{"@type":"koral:term","match":"match:eq","foundry":"myfoundry","layer":"mylayer","key":"mykey1"}`)
	strBuilder.Reset()

	token(&strBuilder, "myfoundry", "mylayer", []string{"mykey1", "mykey2"})
	assert.Equal(strBuilder.String(), "{\"@type\":\"koral:token\",\"wrap\":{\"@type\":\"koral:termGroup\",\"relation\":\"relation:and\",\"operation\":\"operation:and\",\"operands\":[{\"@type\":\"koral:term\",\"match\":\"match:eq\",\"foundry\":\"myfoundry\",\"layer\":\"mylayer\",\"key\":\"mykey1\"},{\"@type\":\"koral:term\",\"match\":\"match:eq\",\"foundry\":\"myfoundry\",\"layer\":\"mylayer\",\"key\":\"mykey2\"}]}}")
	strBuilder.Reset()

	token(&strBuilder, "myfoundry", "mylayer", []string{"mykey2"})
	assert.Equal(strBuilder.String(), "{\"@type\":\"koral:token\",\"wrap\":{\"@type\":\"koral:term\",\"match\":\"match:eq\",\"foundry\":\"myfoundry\",\"layer\":\"mylayer\",\"key\":\"mykey2\"}}")
	strBuilder.Reset()
}

/*
		jsonStr := `{
		"query": {
			"@type": "koral:operation",
			"operators": [
				{
					"@type": "koral:term",
					"key": "example1"
				},
				{
					"@type": "koral:term",
					"key": "example2"
				},
				{
					"@type": "koral:operation",
					"operators": [
						{
							"@type": "koral:term",
							"key": "nested"
						}
					]
				}
			]
		}
	}`
*/

func TestTermReplacement(t *testing.T) {

	assert := assert.New(t)

	// case1: 1 -> 1 the term is wrapped
	testStr := replaceWrappedTerm("{\"@type\":\"koral:term\",\"match\":\"match:eq\",\"foundry\":\"myfoundry\",\"layer\":\"mylayer\",\"key\":\"mykey\"}", "myfoundry2", "mylayer2", "mykey2")
	assert.Equal(testStr, "{\"@type\":\"koral:term\",\"match\":\"match:eq\",\"foundry\":\"myfoundry2\",\"layer\":\"mylayer2\",\"key\":\"mykey2\"}")

	// case2: 1 -> 1 the term is an operand in a termGroup with the same relation/operation
	// case3: 1 -> 1 the term is an operand in a termGroup with a different relation/operation
	testStr = replaceGroupedTerm(
		"{\"@type\":\"koral:termGroup\",\"relation\":\"relation:and\",\"operation\":\"operation:and\",\"operands\":[{\"@type\":\"koral:term\",\"match\":\"match:eq\",\"foundry\":\"myfoundry\",\"layer\":\"mylayer\",\"key\":\"mykey\"},{\"@type\":\"koral:term\",\"match\":\"match:eq\",\"foundry\":\"myfoundry\",\"layer\":\"mylayer\",\"key\":\"mykey2\"}]}",
		[]int{0},
		"myfoundryX",
		"mylayerX",
		"mykeyX",
	)
	assert.Equal(testStr, "{\"@type\":\"koral:termGroup\",\"relation\":\"relation:and\",\"operation\":\"operation:and\",\"operands\":[{\"@type\":\"koral:term\",\"match\":\"match:eq\",\"foundry\":\"myfoundryX\",\"layer\":\"mylayerX\",\"key\":\"mykeyX\"},{\"@type\":\"koral:term\",\"match\":\"match:eq\",\"foundry\":\"myfoundry\",\"layer\":\"mylayer\",\"key\":\"mykey2\"}]}")

	testStr = replaceGroupedTerm(
		"{\"@type\":\"koral:termGroup\",\"relation\":\"relation:and\",\"operation\":\"operation:and\",\"operands\":[{\"@type\":\"koral:term\",\"match\":\"match:ne\",\"foundry\":\"myfoundry\",\"layer\":\"mylayer\",\"key\":\"mykey\"},{\"@type\":\"koral:term\",\"match\":\"match:eq\",\"foundry\":\"myfoundry\",\"layer\":\"mylayer\",\"key\":\"mykey2\"}]}",
		[]int{0},
		"myfoundryX",
		"mylayerX",
		"mykeyX",
	)
	assert.Equal(testStr, "{\"@type\":\"koral:termGroup\",\"relation\":\"relation:and\",\"operation\":\"operation:and\",\"operands\":[{\"@type\":\"koral:term\",\"match\":\"match:ne\",\"foundry\":\"myfoundryX\",\"layer\":\"mylayerX\",\"key\":\"mykeyX\"},{\"@type\":\"koral:term\",\"match\":\"match:eq\",\"foundry\":\"myfoundry\",\"layer\":\"mylayer\",\"key\":\"mykey2\"}]}")

	// case4: n -> 1 the term is an operand in a termGroup with the same relation/operation
	testStr = replaceGroupedTerm(
		"{\"@type\":\"koral:termGroup\",\"relation\":\"relation:and\",\"operation\":\"operation:and\",\"operands\":[{\"@type\":\"koral:term\",\"match\":\"match:ne\",\"foundry\":\"myfoundry\",\"layer\":\"mylayer\",\"key\":\"mykey\"},{\"@type\":\"koral:term\",\"match\":\"match:eq\",\"foundry\":\"myfoundry\",\"layer\":\"mylayer\",\"key\":\"mykey2\"}]}",
		[]int{0, 1},
		"myfoundryX",
		"mylayerX",
		"mykeyX",
	)
	assert.Equal(testStr, "{\"@type\":\"koral:termGroup\",\"relation\":\"relation:and\",\"operation\":\"operation:and\",\"operands\":[{\"@type\":\"koral:term\",\"match\":\"match:ne\",\"foundry\":\"myfoundryX\",\"layer\":\"mylayerX\",\"key\":\"mykeyX\"}]}")

	// case5: n -> 1 the term is an operand in a termGroup with a different relation/operation
	// case6: 1 -> n the term is wrapped
	// case7: 1 -> n the term is an operand in a termGroup with the same relation/operation
	// case8: 1 -> n the term is an operand in a termGroup with a different relation/operation
	// case9: n -> n the term is an operand in a termGroup with the same relation/operation
	// case10: n -> n the term is an operand in a termGroup with a different relation/operation
}
