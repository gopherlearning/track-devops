package web

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateCryptoKeys(t *testing.T) {
	assert.NoError(t, emulateError(nil, 0))
	assert.Error(t, emulateError(errors.New("blank"), 0))
	emulatedError = "must error 0"
	assert.Error(t, emulateError(nil, 0))
	emulatedError = ""

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
			args:    args{keyPath: fmt.Sprintf("%s%sgoTestPem", os.TempDir(), string(os.PathSeparator))},
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
			args:    args{keyPath: fmt.Sprintf("%s%sgoTestPem", os.TempDir(), string(os.PathSeparator))},
			wantErr: true,
			env:     "error os.WriteFile public 4",
		},
		{
			name:    "error os.WriteFile private",
			args:    args{keyPath: "/tmp12345"},
			wantErr: true,
			env:     "error ",
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
