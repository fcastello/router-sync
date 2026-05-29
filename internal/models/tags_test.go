package models

import (
	"reflect"
	"testing"
)

func TestNormalizeTags(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"empty", nil, []string{}},
		{"trim dedupe sort", []string{" b ", "a", "a", "", "b"}, []string{"a", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeTags(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NormalizeTags() = %v, want %v", got, tt.want)
			}
		})
	}
}
