package termmapper

/*
$(	=>	PUNCT	PunctType=Brck	``, '', *RRB*, *LRB*, -
$,	=>	PUNCT	PunctType=Comm	,
$.	=>	PUNCT	PunctType=Peri	., :, ?, ;, !
ADJA	=>	ADJ	_	neuen, neue, deutschen, ersten, anderen
ADJD	=>	ADJ	Variant=Short	gut, rund, knapp, deutlich, möglich
ADV	=>	ADV	_	auch, nur, noch, so, aber
APPO	=>	ADP	AdpType=Post	zufolge, nach, gegenüber, wegen, über
APPR	=>	ADP	AdpType=Prep	in, von, mit, für, auf
APPRART	=>	ADP	AdpType=Prep|PronType=Art	im, am, zum, zur, vom
APZR	=>	ADP	AdpType=Circ	an, hinaus, aus, her, heraus
ART	=>	DET	PronType=Art	der, die, den, des, das
CARD	=>	NUM	NumType=Card	000, zwei, drei, vier, fünf
FM	=>	X	Foreign=Yes	New, of, de, Times, the
ITJ	=>	INTJ	_	naja, Ach, äh, Na, piep
KOKOM	=>	CCONJ	ConjType=Comp	als, wie, denn, wir
KON	=>	CCONJ	_	und, oder, sondern, sowie, aber
KOUI	=>	SCONJ	_	um, ohne, statt, anstatt, Ums
KOUS	=>	SCONJ	_	daß, wenn, weil, ob, als
NE	=>	PROPN	_	SPD, Deutschland, USA, dpa, Bonn
NN	=>	NOUN	_	Prozent, Mark, Millionen, November, Jahren
PAV	=>	ADV	PronType=Dem
PDAT	=>	DET	PronType=Dem	dieser, diese, diesem, dieses, diesen
PDS	=>	PRON	PronType=Dem	das, dies, die, diese, der
PIAT	=>	DET	PronType=Ind,Neg,Tot	keine, mehr, alle, kein, beiden
PIDAT	=>	DET	AdjType=Pdt|PronType=Ind,Neg,Tot
PIS	=>	PRON	PronType=Ind,Neg,Tot	man, allem, nichts, alles, mehr
PPER	=>	PRON	PronType=Prs	es, sie, er, wir, ich
PPOSAT	=>	DET	Poss=Yes|PronType=Prs	ihre, seine, seiner, ihrer, ihren
PPOSS	=>	PRON	Poss=Yes|PronType=Prs	ihren, Seinen, seinem, unsrigen, meiner
PRELAT	=>	DET	PronType=Rel	deren, dessen, die
PRELS	=>	PRON	PronType=Rel	die, der, das, dem, denen
PRF	=>	PRON	PronType=Prs|Reflex=Yes	sich, uns, mich, mir, dich
PTKA	=>	PART	_	zu, am, allzu, Um
PTKANT	=>	PART	PartType=Res	nein, ja, bitte, Gewiß, Also
PTKNEG	=>	PART	Polarity=Neg	nicht
PTKVZ	=>	ADP	PartType=Vbp	an, aus, ab, vor, auf
PTKZU	=>	PART	PartType=Inf	zu, zur, zum
PWAT	=>	DET	PronType=Int	welche, welchen, welcher, wie, welchem
PWAV	=>	ADV	PronType=Int	wie, wo, warum, wobei, wonach
PWS	=>	PRON	PronType=Int	was, wer, wem, wen, welches
TRUNC	=>	X	Hyph=Yes	Staats-, Industrie-, Finanz-, Öl-, Lohn-
VAFIN	=>	AUX	Mood=Ind|VerbForm=Fin	ist, hat, wird, sind, sei
VAIMP	=>	AUX	Mood=Imp|VerbForm=Fin	Seid, werde, Sei
VAINF	=>	AUX	VerbForm=Inf	werden, sein, haben, worden, Dabeisein
VAPP	=>	AUX	Aspect=Perf|VerbForm=Part	worden, gewesen, geworden, gehabt, werden
VMFIN	=>	VERB	Mood=Ind|VerbForm=Fin|VerbType=Mod	kann, soll, will, muß, sollen
VMINF	=>	VERB	VerbForm=Inf|VerbType=Mod	können, müssen, wollen, dürfen, sollen
VMPP	=>	VERB	Aspect=Perf|VerbForm=Part|VerbType=Mod	gewollt
VVFIN	=>	VERB	Mood=Ind|VerbForm=Fin	sagte, gibt, geht, steht, kommt
VVIMP	=>	VERB	Mood=Imp|VerbForm=Fin	siehe, sprich, schauen, Sagen, gestehe
VVINF	=>	VERB	VerbForm=Inf	machen, lassen, bleiben, geben, bringen
VVIZU	=>	VERB	VerbForm=Inf	einzusetzen, durchzusetzen, aufzunehmen, abzubauen, umzusetzen
VVPP	=>	VERB	Aspect=Perf|VerbForm=Part	gemacht, getötet, gefordert, gegeben, gestellt
XY	=>	X	_	dpa, ap, afp, rtr, wb
*/

import (
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

/*
import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

var mapping = map[string]string{
	"$(":"PUNCT",
}

// Recursive function to turn the UPos query into a STTS query
func koralRewriteUpos2Stts(koralquery interface{}) interface{} {
	switch v := koralquery.(type) {
	case map[string]interface{}:
		// Check for '@type' key and act accordingly
		if typ, ok := v["@type"].(string); ok {
			switch typ {
			case "koral:term":

        // Modify the key to use STTS
// This may require to turn object into a koral:token with terms like:


				if key, ok := v["key"].(string); ok {
					v["key"] = "hallo-" + key
				}
			case "operation":
				// Handle the 'operators' key by recursively modifying each operator
				if operators, ok := v["operators"].([]interface{}); ok {
					for i, operator := range operators {
						operators[i] = modifyJSON(operator)
					}
					v["operators"] = operators
				}
			}
		}
		// Recursively modify any nested maps
		for k, val := range v {
			v[k] = modifyJSON(val)
		}
		return v
	case []interface{}:
		// Recursively modify elements of arrays
		for i, item := range v {
			v[i] = modifyJSON(item)
		}
		return v
	}
	return koralquery
}

func main() {
	// Sample JSON input string
	jsonStr := `{
		"@type": "operation",
		"operators": [
			{
				"@type": "term",
				"key": "example1"
			},
			{
				"@type": "term",
				"key": "example2"
			},
			{
				"@type": "operation",
				"operators": [
					{
						"@type": "term",
						"key": "nested"
					}
				]
			}
		]
	}`

	// Parse the JSON string into a generic interface{}
	var data interface{}
	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		log.Fatal("Error unmarshaling JSON:", err)
	}

	// Modify the JSON structure recursively
	modifiedData := modifyJSON(data)

	// Marshal the modified data back into a JSON string
	modifiedJSON, err := json.MarshalIndent(modifiedData, "", "  ")
	if err != nil {
		log.Fatal("Error marshaling JSON:", err)
	}

	// Output the modified JSON string
	fmt.Println(string(modifiedJSON))
}



func turnupostostts(json string, targetFoundry string, targetLayer string) {
	if targetLayer == "" {
		targetLayer = "p"
	}

  ldType := "@type"

  if ldType == "koral:span" {
    next
  }
  if ldType == "koral:term"  {
      if foundry == if layer === key -> rewrite
  }

	// Iterate through the query and whenever a term is requested without a foundry, and without a layser or layer p,
  // change the key following the mapping


}

func addupostooutput(json string, reffoundry string, foundry string) {
	// https://universaldependencies.org/tagset-conversion/de-stts-uposf.html
	// Iterate through all matches and add to all xml snippets a line of foundry

}

*/

type Term struct {
	Foundry string
	Layer   string
	Key     string
}

func Hui() string {
	return "test"
}

func Map2(json []byte) string {
	/*
		result := gjson.GetBytes(json, "query")
		var raw []byte
		if result.Index > 0 {
			raw = json[result.Index:result.Index+len(result.Raw)]
		} else {
			raw = []byte(result.Raw)
		}

		if result.IsObject() {
			koralType := gjson.GetBytes(raw, "@type").String()
			switch koralType {
				case "koral:term":

			}
		}
	*/

	koralObj := gjson.ParseBytes(json)

	switch koralObj.Get("@type").String() {
	case "koral:term":
		{
			if koralObj.Get("value").String() == "KOKOM" {
				// TODO: Turn this in a token, if it isn't already!
				newJson, _ := sjson.Set(string(json), "value", "CCONJ")
				return newJson
			}
		}

	case "koral:operation":
		{

		}

	}
	/*

		var raw []byte
		if result.Index > 0 {
			raw = json[result.Index:result.Index+len(result.Raw)]
		} else {
			raw = []byte(result.Raw)
		}
	*/
	return "jj"
}

// token writes a token to the string builder
func token(strBuilder *strings.Builder, foundry string, layer string, keys []string) {
	strBuilder.WriteString(`{"@type":"koral:token","wrap":`)
	if len(keys) > 1 {
		termGroup(strBuilder, foundry, layer, keys)
	} else {
		term(strBuilder, foundry, layer, keys[0], true)
	}
	strBuilder.WriteString(`}`)
}

// termGroup writes a termGroup to the string builder
func termGroup(strBuilder *strings.Builder, foundry string, layer string, keys []string) {
	strBuilder.WriteString(`{"@type":"koral:termGroup","relation":"relation:and","operation":"operation:and","operands":[`)
	for i, key := range keys {
		term(strBuilder, foundry, layer, key, true) // temporary
		if i < len(keys)-1 {
			strBuilder.WriteString(",")
		}
	}
	strBuilder.WriteString(`]}`)
}

// termGroup2 writes a termGroup to the string builder
func termGroup2(strBuilder *strings.Builder, terms []Term, positive bool) {
	strBuilder.WriteString(`{"@type":"koral:termGroup",`)

	if positive {
		strBuilder.WriteString(`"relation":"relation:and","operation":"operation:and",`)
	} else {
		strBuilder.WriteString(`"relation":"relation:or","operation":"operation:or",`)
	}

	strBuilder.WriteString(`"operands":[`)
	for i, term := range terms {
		term2(strBuilder, term, positive)
		if i < len(terms)-1 {
			strBuilder.WriteString(",")
		}
	}
	strBuilder.WriteString(`]}`)
}

// term writes a term to the string builder
func term(strBuilder *strings.Builder, foundry string, layer string, key string, match bool) {

	// TODO: May have ne!!!!
	strBuilder.WriteString(`{"@type":"koral:term","match":"match:`)
	if match {
		strBuilder.WriteString("eq")
	} else {
		strBuilder.WriteString("ne")
	}
	strBuilder.WriteString(`","foundry":"`)
	strBuilder.WriteString(foundry)
	strBuilder.WriteString(`","layer":"`)
	strBuilder.WriteString(layer)
	strBuilder.WriteString(`","key":"`)
	strBuilder.WriteString(key)
	strBuilder.WriteString(`"}`)
}

// term writes a term to the string builder
func term2(strBuilder *strings.Builder, term Term, match bool) {

	// TODO: May have ne!!!!
	strBuilder.WriteString(`{"@type":"koral:term","match":"match:`)
	if match {
		strBuilder.WriteString("eq")
	} else {
		strBuilder.WriteString("ne")
	}
	strBuilder.WriteString(`","foundry":"`)
	strBuilder.WriteString(term.Foundry)
	strBuilder.WriteString(`","layer":"`)
	strBuilder.WriteString(term.Layer)
	strBuilder.WriteString(`","key":"`)
	strBuilder.WriteString(term.Key)
	strBuilder.WriteString(`"}`)
}

func flatten() {

	// if a termGroup isan operand in a termGroup with the same relation/operation:
	// flatten the termGroup into the parent termGroup

	// if a termGroup has only a single term, remove the group
}

func replaceWrappedTerms(jsonString string, terms []Term) string {
	var err error

	if len(terms) == 1 {
		jsonString, err = sjson.Set(jsonString, "foundry", terms[0].Foundry)
		if err != nil {
			log.Error().Err(err).Msg("Error setting foundry")
		}
		jsonString, err = sjson.Set(jsonString, "layer", terms[0].Layer)
		if err != nil {
			log.Error().Err(err).Msg("Error setting layer")
		}
		jsonString, err = sjson.Set(jsonString, "key", terms[0].Key)
		if err != nil {
			log.Error().Err(err).Msg("Error setting key")
		}

		return jsonString
	}

	matchop := gjson.Get(jsonString, "match").String()

	/*
		foundry := gjson.Get(jsonString, "foundry").String()
		layer := gjson.Get(jsonString, "layer").String()
		key := gjson.Get(jsonString, "key").String()
		term := Term{foundry, layer, key}


		terms = append(terms, term)
	*/

	var strBuilder strings.Builder
	if matchop == "match:ne" {
		termGroup2(&strBuilder, terms, false)
	} else {
		termGroup2(&strBuilder, terms, true)
	}

	return strBuilder.String()

}

func replaceGroupedTerm(jsonString string, op []int, foundry string, layer string, key string) string {
	var err error

	strInt := "operands." + strconv.Itoa(op[0]) + "."
	jsonString, err = sjson.Set(jsonString, strInt+"foundry", foundry)
	if err != nil {
		log.Error().Err(err).Msg("Error setting foundry")
	}
	jsonString, err = sjson.Set(jsonString, strInt+"layer", layer)
	if err != nil {
		log.Error().Err(err).Msg("Error setting layer")
	}
	jsonString, err = sjson.Set(jsonString, strInt+"key", key)
	if err != nil {
		log.Error().Err(err).Msg("Error setting key")
	}

	if len(op) > 1 {
		for i := 1; i < len(op); i++ {
			jsonString, err = sjson.Delete(jsonString, "operands."+strconv.Itoa(op[i]))
			if err != nil {
				log.Error().Err(err).Msg("Error deleting operand")
			}
		}
	}

	return jsonString
}

/*
func replaceTermWithToken(jsonString string) string {
	// Replace the term with the token
	replacedString, err := sjson.Set(jsonString, "wrap.operands.0", token())
	if err != nil {
		return jsonString // Return the original string in case of an error
	}
	return replacedString

// case1: 1 -> 1 the term is an operand in a termGroup with the same relation/operation
// case2: 1 -> 1 the term is wrapped
// case3: 1 -> 1 the term is an operand in a termGroup with a different relation/operation
// case4: n -> 1 the term is an operand in a termGroup with the same relation/operation
// case5: n -> 1 the term is wrapped
// case6: n -> 1 the term is an operand in a termGroup with a different relation/operation
// case7: 1 -> n the term is an operand in a termGroup with the same relation/operation
// case8: 1 -> n the term is wrapped
// case9: 1 -> n the term is an operand in a termGroup with a different relation/operation
	}
*/

func Map(jsonStr string) string {

	obj := gjson.Get(jsonStr, "query")

	// value := gjson.Get(json, "name.last")

	/*

	   	// Modify the JSON structure recursively
	   	modifiedData := modifyJSON(ast.NewAny(data))

	   // Marshal the modified data back into a JSON string
	   	modifiedJSON, err := sonic.MarshalString(modifiedData)

	   	// Parse the JSON string into a generic interface{}
	   	var data interface{}

	   	err := sonic.UnmarshalString(jsonStr, data)

	   	if err != nil {
	   		log.Fatal("Error unmarshaling JSON:", err)
	   		return ""
	   	}



	   	if err != nil {
	   		log.Fatal("Error marshaling JSON:", err)
	   	}
	*/
	// Output the modified JSON string
	return obj.String() //modifyJSON(obj)
}

// Recursive function to modify JSON using Sonic library
//func modifyJSON(data gjson.Result) string {

// Check if data is a map
// if data.IsObject() {
/*
	dataMap := data.Map()

	koralType := dataMap["@type"].String()

	// Look for @type key

	switch koralType {
	case "koral:term":
		// Modify the key by adding 'hallo-' prefix

		// sjson.SetRaw(data.String())
		sjson.Set(data.Path(data.Bytes()), "key", "hallo-"+dataMap["key"].String())

		dataMap["key"] = "hallo-" + dataMap["key"].String()
		/*
			if key, found := data.GetString("key"); found {
				data.Set("key", "hallo-"+key)
			}
*/
/*
	case "koral:operation":
			// Handle the 'operators' key by recursively modifying each operator
			if operators, found := data.GetArray("operators"); found {
				for i := range operators {
					operators[i] = modifyJSON(operators[i])
				}
				data.Set("operators", operators)
			}
		}*/
/*
	// Recursively modify any nested objects
	data.ForEach(func(k string, v sonic.Any) {
		data.Set(k, modifyJSON(v))
	})
*/
//}
// Handle arrays by modifying elements recursively
/*
	if data.IsArray() {
		for i := range data.GetArray() {
			data.Set(i, modifyJSON(data.GetArray()[i]))
		}
	}
*/
/*
	return data
}
*/
