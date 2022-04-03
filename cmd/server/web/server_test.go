package web

import (
	"reflect"
	"testing"
)

func TestServer(t *testing.T) {
	tests := []struct {
		name   string
		listen string
	}{
		{
			name:   "Создание сервера",
			listen: "127.0.0.1:12328",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewServer(tt.listen, nil)
			if !reflect.DeepEqual(got, nil) {
				t.Errorf("NewServer() = %v, want %v", got, nil)
			}
		})
	}
}
