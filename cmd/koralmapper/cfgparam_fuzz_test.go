package main

import (
	"reflect"
	"testing"
)

func FuzzParseCfgParam(f *testing.F) {
	f.Add("")
	f.Add("stts-upos:atob")
	f.Add("stts-upos:btoa:opennlp:p:upos:p")
	f.Add("stts-upos:atob;other-mapper:btoa:stts:p:ud:pos")
	f.Add("corpus-map:atob")
	f.Add("unknown-id:atob")
	f.Add("stts-upos:invalid")
	f.Add("stts-upos:atob::::")

	f.Fuzz(func(t *testing.T, raw string) {
		parsed, err := ParseCfgParam(raw, cfgTestLists)
		if err != nil {
			return
		}

		rebuilt := BuildCfgParam(parsed)
		reparsed, err := ParseCfgParam(rebuilt, cfgTestLists)
		if err != nil {
			t.Fatalf("reparse failed for rebuilt cfg %q from raw %q: %v", rebuilt, raw, err)
		}

		if !reflect.DeepEqual(parsed, reparsed) {
			t.Fatalf("round-trip mismatch for raw %q: parsed=%#v reparsed=%#v rebuilt=%q", raw, parsed, reparsed, rebuilt)
		}
	})
}
