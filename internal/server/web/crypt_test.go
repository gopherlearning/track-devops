package web

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCryptoKeys(t *testing.T) {
	assert.NoError(t, emulateError(nil, 0))
	assert.Error(t, emulateError(errors.New("blank"), 0))
	emulatedError = "must error 0"
	assert.Error(t, emulateError(nil, 0))
	emulatedError = ""
	tmp, err := os.CreateTemp(os.TempDir(), "go_test")
	require.NoError(t, err)
	require.FileExists(t, tmp.Name())
	type args struct {
		keyPath string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		env     string
	}{
		{
			name:    "success",
			args:    args{keyPath: tmp.Name()},
			wantErr: false,
			env:     "",
		},
		{
			name:    "error rsa.GenerateKey",
			args:    args{keyPath: ""},
			wantErr: true,
			env:     "error rsa.GenerateKey 1",
		},
		{
			name:    "error pem.Encode public",
			args:    args{keyPath: ""},
			wantErr: true,
			env:     "error pem.Encode 2",
		},
		{
			name:    "error pem.Encode private",
			args:    args{keyPath: ""},
			wantErr: true,
			env:     "error pem.Encode 3",
		},
		{
			name:    "error os.WriteFile public",
			args:    args{keyPath: tmp.Name()},
			wantErr: true,
			env:     "error os.WriteFile public 4",
		},
		{
			name:    "error os.WriteFile private",
			args:    args{keyPath: "/root/aaa/aaa"},
			wantErr: true,
			env:     "error os.WriteFile private 5",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.env) != 0 {
				emulatedError = tt.env
				defer func() { emulatedError = "" }()
			}
			if err := GenerateCryptoKeys(tt.args.keyPath); (err != nil) != tt.wantErr {
				t.Errorf("GenerateCryptoKeys() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
