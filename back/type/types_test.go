package types

import (
	"encoding/json"
	"testing"
)

func TestNullableStringMapsMarshalAndUnmarshalAsJSONStringsOrNull(t *testing.T) {
	value := NullableString("linkedin.com/in/maria")
	payload := map[string]*NullableString{
		"linkedin": &value,
		"area":     nil,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal nullable string map: %v", err)
	}

	if string(raw) != `{"area":null,"linkedin":"linkedin.com/in/maria"}` {
		t.Fatalf("unexpected JSON payload: %s", string(raw))
	}

	var decoded map[string]*NullableString
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("failed to unmarshal nullable string map: %v", err)
	}

	if decoded["area"] != nil {
		t.Fatalf("expected nil area after round-trip, got %#v", decoded["area"])
	}
	if decoded["linkedin"] == nil || string(*decoded["linkedin"]) != "linkedin.com/in/maria" {
		t.Fatalf("unexpected linkedin after round-trip: %#v", decoded["linkedin"])
	}
}
