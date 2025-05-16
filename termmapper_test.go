package termmapper

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// KoralPipe-TermMapping

func TestTokenBuilder(t *testing.T) {

	assert := assert.New(t)

	var strBuilder strings.Builder
	term(&strBuilder, Term{"myfoundry", "mylayer", "mykey1"}, true)
	assert.Equal(strBuilder.String(), `{"@type":"koral:term","match":"match:eq","foundry":"myfoundry","layer":"mylayer","key":"mykey1"}`)
	strBuilder.Reset()

	token(&strBuilder, []Term{{"myfoundry", "mylayer", "mykey1"}, {"myfoundry", "mylayer", "mykey2"}}, true)
	assert.Equal(strBuilder.String(), "{\"@type\":\"koral:token\",\"wrap\":{\"@type\":\"koral:termGroup\",\"relation\":\"relation:and\",\"operation\":\"operation:and\",\"operands\":[{\"@type\":\"koral:term\",\"match\":\"match:eq\",\"foundry\":\"myfoundry\",\"layer\":\"mylayer\",\"key\":\"mykey1\"},{\"@type\":\"koral:term\",\"match\":\"match:eq\",\"foundry\":\"myfoundry\",\"layer\":\"mylayer\",\"key\":\"mykey2\"}]}}")
	strBuilder.Reset()

	token(&strBuilder, []Term{{"myfoundry", "mylayer", "mykey2"}}, true)
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

	// case1: 1 -> 1 the term is wrapped with eq
	// case1: 1 -> 1 the term is wrapped with ne
	// [ADV] -> [ADV]
	testStr := replaceWrappedTerms(
		"{\"@type\":\"koral:term\",\"match\":\"match:eq\",\"foundry\":\"myfoundry\",\"layer\":\"mylayer\",\"key\":\"mykey\"}",
		[]Term{{"myfoundry2",
			"mylayer2",
			"mykey2",
		}},
	)
	assert.Equal(testStr, "{\"@type\":\"koral:term\",\"match\":\"match:eq\",\"foundry\":\"myfoundry2\",\"layer\":\"mylayer2\",\"key\":\"mykey2\"}")

	testStr = replaceWrappedTerms(
		"{\"@type\":\"koral:term\",\"match\":\"match:ne\",\"foundry\":\"myfoundry\",\"layer\":\"mylayer\",\"key\":\"mykey\"}",
		[]Term{{"myfoundry2",
			"mylayer2",
			"mykey2",
		}},
	)
	assert.Equal(testStr, "{\"@type\":\"koral:term\",\"match\":\"match:ne\",\"foundry\":\"myfoundry2\",\"layer\":\"mylayer2\",\"key\":\"mykey2\"}")

	// case2: 1 -> 1 the term is an operand in a termGroup with the same relation/operation
	// [ADV & ...] -> [ADV]
	// case3: 1 -> 1 the term is an operand in a termGroup with a different relation/operation
	// [ADV | ...] -> [ADV]
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
	// [PRON & Poss=yes & PronType=Prs] -> [PPOSAT]
	testStr = replaceGroupedTerm(
		"{\"@type\":\"koral:termGroup\",\"relation\":\"relation:and\",\"operation\":\"operation:and\",\"operands\":[{\"@type\":\"koral:term\",\"match\":\"match:ne\",\"foundry\":\"myfoundry\",\"layer\":\"mylayer\",\"key\":\"mykey\"},{\"@type\":\"koral:term\",\"match\":\"match:eq\",\"foundry\":\"myfoundry\",\"layer\":\"mylayer\",\"key\":\"mykey2\"}]}",
		[]int{0, 1},
		"myfoundryX",
		"mylayerX",
		"mykeyX",
	)
	assert.Equal(testStr, "{\"@type\":\"koral:termGroup\",\"relation\":\"relation:and\",\"operation\":\"operation:and\",\"operands\":[{\"@type\":\"koral:term\",\"match\":\"match:ne\",\"foundry\":\"myfoundryX\",\"layer\":\"mylayerX\",\"key\":\"mykeyX\"}]}")

	// case5: 1 -> n the term is wrapped
	// [PPOSAT] -> [PRON & Poss=yes & PronType=Prs]
	testStr = replaceWrappedTerms(
		"{\"@type\":\"koral:term\",\"match\":\"match:eq\",\"foundry\":\"myfoundry\",\"layer\":\"mylayer\",\"key\":\"mykey\"}",
		[]Term{{
			"myfoundry1",
			"mylayer1",
			"mykey1",
		}, {
			"myfoundry2",
			"mylayer2",
			"mykey2",
		}},
	)
	assert.Equal(testStr, "{\"@type\":\"koral:termGroup\",\"relation\":\"relation:and\",\"operation\":\"operation:and\",\"operands\":[{\"@type\":\"koral:term\",\"match\":\"match:eq\",\"foundry\":\"myfoundry1\",\"layer\":\"mylayer1\",\"key\":\"mykey1\"},{\"@type\":\"koral:term\",\"match\":\"match:eq\",\"foundry\":\"myfoundry2\",\"layer\":\"mylayer2\",\"key\":\"mykey2\"}]}")

	// [!PPOSAT] -> [!PRON | !Poss=yes | !PronType=Prs]
	testStr = replaceWrappedTerms(
		"{\"@type\":\"koral:term\",\"match\":\"match:ne\",\"foundry\":\"myfoundry\",\"layer\":\"mylayer\",\"key\":\"mykey\"}",
		[]Term{{
			"myfoundry1",
			"mylayer1",
			"mykey1",
		}, {
			"myfoundry2",
			"mylayer2",
			"mykey2",
		}},
	)
	assert.Equal(testStr, "{\"@type\":\"koral:termGroup\",\"relation\":\"relation:or\",\"operation\":\"operation:or\",\"operands\":[{\"@type\":\"koral:term\",\"match\":\"match:ne\",\"foundry\":\"myfoundry1\",\"layer\":\"mylayer1\",\"key\":\"mykey1\"},{\"@type\":\"koral:term\",\"match\":\"match:ne\",\"foundry\":\"myfoundry2\",\"layer\":\"mylayer2\",\"key\":\"mykey2\"}]}")

	// case6: 1 -> n the term is an operand in a termGroup with the same relation/operation
	// [PPOSAT & ...] -> [PRON & Poss=yes & PronType=Prs]
	testStr = replaceGroupedTerms(
		"{\"@type\":\"koral:termGroup\",\"relation\":\"relation:and\",\"operation\":\"operation:and\",\"operands\":[{\"@type\":\"koral:term\",\"match\":\"match:eq\",\"foundry\":\"myfoundry\",\"layer\":\"mylayer\",\"key\":\"mykey1\"},{\"@type\":\"koral:term\",\"match\":\"match:eq\",\"foundry\":\"myfoundry\",\"layer\":\"mylayer\",\"key\":\"mykey2\"}]}",
		[]int{0},
		[]Term{{
			"myfoundry3",
			"mylayer3",
			"mykey3",
		}, {
			"myfoundry4",
			"mylayer4",
			"mykey4",
		}},
	)
	assert.Equal(testStr, "{\"@type\":\"koral:termGroup\",\"relation\":\"relation:and\",\"operation\":\"operation:and\",\"operands\":[{\"@type\":\"koral:term\",\"match\":\"match:eq\",\"foundry\":\"myfoundry\",\"layer\":\"mylayer\",\"key\":\"mykey2\"},{\"@type\":\"koral:term\",\"match\":\"match:eq\",\"foundry\":\"myfoundry3\",\"layer\":\"mylayer3\",\"key\":\"mykey3\"},{\"@type\":\"koral:term\",\"match\":\"match:eq\",\"foundry\":\"myfoundry4\",\"layer\":\"mylayer4\",\"key\":\"mykey4\"}]}")

	// case7: 1 -> n the term is an operand in a termGroup with a different relation/operation
	testStr = replaceGroupedTerms(

		"{\"@type\":\"koral:termGroup\",\"relation\":\"relation:and\",\"operation\":\"operation:and\",\"operands\":["+
			"{\"@type\":\"koral:term\",\"match\":\"match:ne\",\"foundry\":\"myfoundry\",\"layer\":\"mylayer\",\"key\":\"mykey1\"},"+
			"{\"@type\":\"koral:term\",\"match\":\"match:eq\",\"foundry\":\"myfoundry\",\"layer\":\"mylayer\",\"key\":\"mykey2\"}"+
			"]}",
		[]int{0},
		[]Term{{
			"myfoundry3",
			"mylayer3",
			"mykey3",
		}, {
			"myfoundry4",
			"mylayer4",
			"mykey4",
		}},
	)

	// TODO: Add a termGroup with reversed signs
	assert.Equal(testStr,

		"{\"@type\":\"koral:termGroup\",\"relation\":\"relation:and\",\"operation\":\"operation:and\",\"operands\":["+

			"{\"@type\":\"koral:term\",\"match\":\"match:eq\",\"foundry\":\"myfoundry\",\"layer\":\"mylayer\",\"key\":\"mykey2\"},"+

			"{\"@type\":\"koral:termGroup\",\"relation\":\"relation:or\",\"operation\":\"operation:or\",\"operands\":["+

			"{\"@type\":\"koral:term\",\"match\":\"match:ne\",\"foundry\":\"myfoundry3\",\"layer\":\"mylayer3\",\"key\":\"mykey3\"},"+
			"{\"@type\":\"koral:term\",\"match\":\"match:ne\",\"foundry\":\"myfoundry4\",\"layer\":\"mylayer4\",\"key\":\"mykey4\"}"+
			"]}"+

			"]"+
			"}")
	// case8: n -> n the term is an operand in a termGroup with the same relation/operation
	// case9: n -> n the term is an operand in a termGroup with a different relation/operation

}
