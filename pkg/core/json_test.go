package core

import (
	"encoding/json"
	"testing"
)

func TestJSONEncode(t *testing.T) {
	tests := []struct {
		name    string
		v       interface{}
		wantErr bool
	}{
		{"valid map", map[string]string{"key": "value"}, false},
		{"valid string", "test", false},
		{"nil value", nil, true},
		{"valid struct", struct{ Name string }{"test"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := JSONEncode(tt.v)
			if (err != nil) != tt.wantErr {
				t.Errorf("JSONEncode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJSONEncode_Pooling(t *testing.T) {
	// Test that encoding works correctly across multiple calls
	data1 := map[string]string{"key1": "value1"}
	data2 := map[string]int{"key2": 42}

	encoded1, err := JSONEncode(data1)
	if err != nil {
		t.Fatalf("JSONEncode() error = %v", err)
	}

	encoded2, err := JSONEncode(data2)
	if err != nil {
		t.Fatalf("JSONEncode() error = %v", err)
	}

	// Verify both encodings are correct using standard json for compatibility check
	var decoded1 map[string]string
	if err := json.Unmarshal(encoded1, &decoded1); err != nil {
		t.Errorf("Failed to decode encoded1: %v", err)
	}
	if decoded1["key1"] != "value1" {
		t.Errorf("decoded1 = %v, want map[key1:value1]", decoded1)
	}

	var decoded2 map[string]int
	if err := json.Unmarshal(encoded2, &decoded2); err != nil {
		t.Errorf("Failed to decode encoded2: %v", err)
	}
	if decoded2["key2"] != 42 {
		t.Errorf("decoded2 = %v, want map[key2:42]", decoded2)
	}
}

func TestJSONDecode(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		v       interface{}
		wantErr bool
	}{
		{"valid json", []byte(`{"key":"value"}`), &map[string]string{}, false},
		{"empty data", []byte{}, &map[string]string{}, true},
		{"nil target", []byte(`{"key":"value"}`), nil, true},
		{"invalid json", []byte(`{invalid}`), &map[string]string{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := JSONDecode(tt.data, tt.v)
			if (err != nil) != tt.wantErr {
				t.Errorf("JSONDecode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJSONEncodeDecode(t *testing.T) {
	original := map[string]interface{}{
		"name":  "test",
		"value": 42,
		"nested": map[string]string{
			"key": "value",
		},
	}

	encoded, err := JSONEncode(original)
	if err != nil {
		t.Fatalf("JSONEncode() error = %v", err)
	}

	var decoded map[string]interface{}
	err = JSONDecode(encoded, &decoded)
	if err != nil {
		t.Fatalf("JSONDecode() error = %v", err)
	}

	if decoded["name"] != original["name"] {
		t.Errorf("decoded name = %v, want %v", decoded["name"], original["name"])
	}
}

func TestJSONEncode_FailFast_NilValue(t *testing.T) {
	_, err := JSONEncode(nil)
	if err == nil {
		t.Error("JSONEncode() should fail-fast with nil value")
	}
	if err != nil {
		if e, ok := err.(*EventBusError); ok {
			if e.Code != "INVALID_INPUT" {
				t.Errorf("Error code = %v, want 'INVALID_INPUT'", e.Code)
			}
		}
	}
}

func TestJSONDecode_FailFast_EmptyData(t *testing.T) {
	var result map[string]string
	err := JSONDecode([]byte{}, &result)
	if err == nil {
		t.Error("JSONDecode() should fail-fast with empty data")
	}
	if err != nil {
		if e, ok := err.(*EventBusError); ok {
			if e.Code != "INVALID_INPUT" {
				t.Errorf("Error code = %v, want 'INVALID_INPUT'", e.Code)
			}
		}
	}
}

func TestJSONDecode_FailFast_NilTarget(t *testing.T) {
	data := []byte(`{"key":"value"}`)
	err := JSONDecode(data, nil)
	if err == nil {
		t.Error("JSONDecode() should fail-fast with nil target")
	}
	if err != nil {
		if e, ok := err.(*EventBusError); ok {
			if e.Code != "INVALID_INPUT" {
				t.Errorf("Error code = %v, want 'INVALID_INPUT'", e.Code)
			}
		}
	}
}

func TestJSONDecode_FailFast_InvalidJSON(t *testing.T) {
	var result map[string]string
	invalidJSON := []byte(`{invalid json}`)
	err := JSONDecode(invalidJSON, &result)
	if err == nil {
		t.Error("JSONDecode() should fail-fast with invalid JSON")
	}
}

func TestJSONEncode_ValidTypes(t *testing.T) {
	tests := []struct {
		name string
		v    interface{}
	}{
		{"string", "test"},
		{"int", 42},
		{"float", 3.14},
		{"bool", true},
		{"slice", []string{"a", "b"}},
		{"map", map[string]int{"a": 1}},
		{"struct", struct{ Name string }{"test"}},
		{"nested map", map[string]interface{}{"nested": map[string]int{"a": 1}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := JSONEncode(tt.v)
			if err != nil {
				t.Errorf("JSONEncode() error = %v for type %T", err, tt.v)
			}
		})
	}
}
