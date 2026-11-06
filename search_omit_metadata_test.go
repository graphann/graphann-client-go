package graphann

import (
	"encoding/json"
	"testing"
)

func TestSearchFilterOmitMetadataJSON(t *testing.T) {
	b, err := json.Marshal(SearchFilter{OmitMetadata: true})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if m["omit_metadata"] != true {
		t.Errorf("omit_metadata = %v, want true", m["omit_metadata"])
	}

	b, _ = json.Marshal(SearchFilter{})
	var d map[string]any
	_ = json.Unmarshal(b, &d)
	if _, ok := d["omit_metadata"]; ok {
		t.Errorf("unset omit_metadata must be omitted, got %s", b)
	}
}
