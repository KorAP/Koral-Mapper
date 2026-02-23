package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCorpusParserSimpleField(t *testing.T) {
	p := NewCorpusParser()
	result, err := p.ParseMapping("textClass=novel <> genre=fiction")
	require.NoError(t, err)

	upper, ok := result.Upper.(*CorpusField)
	require.True(t, ok)
	assert.Equal(t, "textClass", upper.Key)
	assert.Equal(t, "novel", upper.Value)
	assert.Equal(t, "", upper.Match)
	assert.Equal(t, "", upper.Type)

	lower, ok := result.Lower.(*CorpusField)
	require.True(t, ok)
	assert.Equal(t, "genre", lower.Key)
	assert.Equal(t, "fiction", lower.Value)
}

func TestCorpusParserMatchType(t *testing.T) {
	p := NewCorpusParser()
	result, err := p.ParseMapping("pubDate=2020:geq <> yearFrom=2020:geq")
	require.NoError(t, err)

	upper := result.Upper.(*CorpusField)
	assert.Equal(t, "pubDate", upper.Key)
	assert.Equal(t, "2020", upper.Value)
	assert.Equal(t, "geq", upper.Match)

	lower := result.Lower.(*CorpusField)
	assert.Equal(t, "yearFrom", lower.Key)
	assert.Equal(t, "2020", lower.Value)
	assert.Equal(t, "geq", lower.Match)
}

func TestCorpusParserValueType(t *testing.T) {
	p := NewCorpusParser()
	result, err := p.ParseMapping("pubDate=2020-01#date <> year=2020#string")
	require.NoError(t, err)

	upper := result.Upper.(*CorpusField)
	assert.Equal(t, "pubDate", upper.Key)
	assert.Equal(t, "2020-01", upper.Value)
	assert.Equal(t, "", upper.Match)
	assert.Equal(t, "date", upper.Type)

	lower := result.Lower.(*CorpusField)
	assert.Equal(t, "year", lower.Key)
	assert.Equal(t, "2020", lower.Value)
	assert.Equal(t, "", lower.Match)
	assert.Equal(t, "string", lower.Type)
}

func TestCorpusParserMatchAndType(t *testing.T) {
	p := NewCorpusParser()
	result, err := p.ParseMapping("pubDate=2020:geq#date <> year=2020:geq#string")
	require.NoError(t, err)

	upper := result.Upper.(*CorpusField)
	assert.Equal(t, "pubDate", upper.Key)
	assert.Equal(t, "2020", upper.Value)
	assert.Equal(t, "geq", upper.Match)
	assert.Equal(t, "date", upper.Type)

	lower := result.Lower.(*CorpusField)
	assert.Equal(t, "year", lower.Key)
	assert.Equal(t, "2020", lower.Value)
	assert.Equal(t, "geq", lower.Match)
	assert.Equal(t, "string", lower.Type)
}

func TestCorpusParserRegex(t *testing.T) {
	p := NewCorpusParser()
	result, err := p.ParseMapping("textClass=wissenschaft.*#regex <> genre=science")
	require.NoError(t, err)

	upper := result.Upper.(*CorpusField)
	assert.Equal(t, "textClass", upper.Key)
	assert.Equal(t, "wissenschaft.*", upper.Value)
	assert.Equal(t, "regex", upper.Type)

	lower := result.Lower.(*CorpusField)
	assert.Equal(t, "genre", lower.Key)
	assert.Equal(t, "science", lower.Value)
}

func TestCorpusParserANDGroup(t *testing.T) {
	p := NewCorpusParser()
	result, err := p.ParseMapping("(textClass=novel & pubDate=2020) <> genre=fiction")
	require.NoError(t, err)

	group, ok := result.Upper.(*CorpusGroup)
	require.True(t, ok)
	assert.Equal(t, "and", group.Operation)
	require.Len(t, group.Operands, 2)

	f1 := group.Operands[0].(*CorpusField)
	assert.Equal(t, "textClass", f1.Key)
	assert.Equal(t, "novel", f1.Value)

	f2 := group.Operands[1].(*CorpusField)
	assert.Equal(t, "pubDate", f2.Key)
	assert.Equal(t, "2020", f2.Value)

	lower := result.Lower.(*CorpusField)
	assert.Equal(t, "genre", lower.Key)
	assert.Equal(t, "fiction", lower.Value)
}

func TestCorpusParserORGroup(t *testing.T) {
	p := NewCorpusParser()
	result, err := p.ParseMapping("(textClass=novel | textClass=fiction) <> genre=fiction")
	require.NoError(t, err)

	group, ok := result.Upper.(*CorpusGroup)
	require.True(t, ok)
	assert.Equal(t, "or", group.Operation)
	require.Len(t, group.Operands, 2)

	f1 := group.Operands[0].(*CorpusField)
	assert.Equal(t, "textClass", f1.Key)
	assert.Equal(t, "novel", f1.Value)

	f2 := group.Operands[1].(*CorpusField)
	assert.Equal(t, "textClass", f2.Key)
	assert.Equal(t, "fiction", f2.Value)
}

func TestCorpusParserNestedGroup(t *testing.T) {
	p := NewCorpusParser()
	result, err := p.ParseMapping("(a=1 & (b=2 | c=3)) <> d=4")
	require.NoError(t, err)

	outer, ok := result.Upper.(*CorpusGroup)
	require.True(t, ok)
	assert.Equal(t, "and", outer.Operation)
	require.Len(t, outer.Operands, 2)

	f1 := outer.Operands[0].(*CorpusField)
	assert.Equal(t, "a", f1.Key)
	assert.Equal(t, "1", f1.Value)

	inner, ok := outer.Operands[1].(*CorpusGroup)
	require.True(t, ok)
	assert.Equal(t, "or", inner.Operation)
	require.Len(t, inner.Operands, 2)

	f2 := inner.Operands[0].(*CorpusField)
	assert.Equal(t, "b", f2.Key)
	f3 := inner.Operands[1].(*CorpusField)
	assert.Equal(t, "c", f3.Key)
}

func TestCorpusParserSingleToGroup(t *testing.T) {
	p := NewCorpusParser()
	result, err := p.ParseMapping("textClass=novel <> (genre=fiction & type=book)")
	require.NoError(t, err)

	_, ok := result.Upper.(*CorpusField)
	require.True(t, ok)

	group, ok := result.Lower.(*CorpusGroup)
	require.True(t, ok)
	assert.Equal(t, "and", group.Operation)
	require.Len(t, group.Operands, 2)
}

func TestCorpusParserGroupToSingle(t *testing.T) {
	p := NewCorpusParser()
	result, err := p.ParseMapping("(genre=fiction & type=book) <> textClass=novel")
	require.NoError(t, err)

	_, ok := result.Upper.(*CorpusGroup)
	require.True(t, ok)

	_, ok = result.Lower.(*CorpusField)
	require.True(t, ok)
}

func TestCorpusParserErrors(t *testing.T) {
	p := NewCorpusParser()

	_, err := p.ParseMapping("textClass=novel")
	assert.Error(t, err, "missing <> separator")

	_, err = p.ParseMapping(" <> genre=fiction")
	assert.Error(t, err, "empty left side")

	_, err = p.ParseMapping("textClass=novel <> ")
	assert.Error(t, err, "empty right side")

	_, err = p.ParseMapping("invalidfield <> genre=fiction")
	assert.Error(t, err, "missing = in field")
}

func TestCorpusParserThreeOperandGroup(t *testing.T) {
	p := NewCorpusParser()
	result, err := p.ParseMapping("(a=1 & b=2 & c=3) <> d=4")
	require.NoError(t, err)

	group, ok := result.Upper.(*CorpusGroup)
	require.True(t, ok)
	assert.Equal(t, "and", group.Operation)
	require.Len(t, group.Operands, 3)
}

func TestCorpusParserWhitespaceHandling(t *testing.T) {
	p := NewCorpusParser()
	result, err := p.ParseMapping("  textClass=novel  <>  genre=fiction  ")
	require.NoError(t, err)

	upper := result.Upper.(*CorpusField)
	assert.Equal(t, "textClass", upper.Key)
	assert.Equal(t, "novel", upper.Value)
}

func TestCorpusFieldToJSON(t *testing.T) {
	field := &CorpusField{Key: "textClass", Value: "novel"}
	json := field.ToJSON()
	assert.Equal(t, "koral:doc", json["@type"])
	assert.Equal(t, "textClass", json["key"])
	assert.Equal(t, "novel", json["value"])
	assert.Equal(t, "match:eq", json["match"])
	assert.Equal(t, "type:string", json["type"])
}

func TestCorpusFieldToJSONWithMatchAndType(t *testing.T) {
	field := &CorpusField{Key: "pubDate", Value: "2020", Match: "geq", Type: "date"}
	json := field.ToJSON()
	assert.Equal(t, "koral:doc", json["@type"])
	assert.Equal(t, "pubDate", json["key"])
	assert.Equal(t, "2020", json["value"])
	assert.Equal(t, "match:geq", json["match"])
	assert.Equal(t, "type:date", json["type"])
}

func TestCorpusGroupToJSON(t *testing.T) {
	group := &CorpusGroup{
		Operation: "and",
		Operands: []CorpusNode{
			&CorpusField{Key: "genre", Value: "fiction"},
			&CorpusField{Key: "type", Value: "book"},
		},
	}
	json := group.ToJSON()
	assert.Equal(t, "koral:docGroup", json["@type"])
	assert.Equal(t, "operation:and", json["operation"])
	operands, ok := json["operands"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, operands, 2)
	assert.Equal(t, "genre", operands[0]["key"])
	assert.Equal(t, "type", operands[1]["key"])
}

func TestCorpusFieldClone(t *testing.T) {
	f := &CorpusField{Key: "a", Value: "b", Match: "eq", Type: "string"}
	c := f.Clone().(*CorpusField)
	assert.Equal(t, f.Key, c.Key)
	assert.Equal(t, f.Value, c.Value)
	c.Key = "changed"
	assert.NotEqual(t, f.Key, c.Key)
}

func TestCorpusGroupClone(t *testing.T) {
	g := &CorpusGroup{
		Operation: "and",
		Operands: []CorpusNode{
			&CorpusField{Key: "a", Value: "1"},
			&CorpusField{Key: "b", Value: "2"},
		},
	}
	c := g.Clone().(*CorpusGroup)
	assert.Equal(t, g.Operation, c.Operation)
	require.Len(t, c.Operands, 2)
	c.Operands[0].(*CorpusField).Key = "changed"
	assert.NotEqual(t, g.Operands[0].(*CorpusField).Key, "changed")
}

func TestCorpusParserBareValue(t *testing.T) {
	p := &CorpusParser{AllowBareValues: true}
	result, err := p.ParseMapping("Entertainment <> (technik-industrie & edv-elektronik)")
	require.NoError(t, err)

	upper, ok := result.Upper.(*CorpusField)
	require.True(t, ok)
	assert.Equal(t, "", upper.Key)
	assert.Equal(t, "Entertainment", upper.Value)

	group, ok := result.Lower.(*CorpusGroup)
	require.True(t, ok)
	assert.Equal(t, "and", group.Operation)
	require.Len(t, group.Operands, 2)

	f1 := group.Operands[0].(*CorpusField)
	assert.Equal(t, "", f1.Key)
	assert.Equal(t, "technik-industrie", f1.Value)

	f2 := group.Operands[1].(*CorpusField)
	assert.Equal(t, "", f2.Key)
	assert.Equal(t, "edv-elektronik", f2.Value)
}

func TestCorpusParserBareValueORGroup(t *testing.T) {
	p := &CorpusParser{AllowBareValues: true}
	result, err := p.ParseMapping("Entertainment <> ((kultur & musik) | (kultur & film))")
	require.NoError(t, err)

	upper, ok := result.Upper.(*CorpusField)
	require.True(t, ok)
	assert.Equal(t, "Entertainment", upper.Value)

	group, ok := result.Lower.(*CorpusGroup)
	require.True(t, ok)
	assert.Equal(t, "or", group.Operation)
	require.Len(t, group.Operands, 2)

	and1, ok := group.Operands[0].(*CorpusGroup)
	require.True(t, ok)
	assert.Equal(t, "and", and1.Operation)
	assert.Equal(t, "kultur", and1.Operands[0].(*CorpusField).Value)
	assert.Equal(t, "musik", and1.Operands[1].(*CorpusField).Value)

	and2, ok := group.Operands[1].(*CorpusGroup)
	require.True(t, ok)
	assert.Equal(t, "and", and2.Operation)
	assert.Equal(t, "kultur", and2.Operands[0].(*CorpusField).Value)
	assert.Equal(t, "film", and2.Operands[1].(*CorpusField).Value)
}

func TestCorpusParserBareValueDisabledByDefault(t *testing.T) {
	p := NewCorpusParser()
	_, err := p.ParseMapping("Entertainment <> genre=fiction")
	assert.Error(t, err, "bare values should fail without AllowBareValues")
}

func TestCorpusParserBareValueMixedWithKeyed(t *testing.T) {
	p := &CorpusParser{AllowBareValues: true}
	result, err := p.ParseMapping("Entertainment <> genre=fiction")
	require.NoError(t, err)

	upper := result.Upper.(*CorpusField)
	assert.Equal(t, "", upper.Key)
	assert.Equal(t, "Entertainment", upper.Value)

	lower := result.Lower.(*CorpusField)
	assert.Equal(t, "genre", lower.Key)
	assert.Equal(t, "fiction", lower.Value)
}
