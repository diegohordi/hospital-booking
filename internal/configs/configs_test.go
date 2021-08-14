package configs

import (
	"testing"
)

func TestLoad(t *testing.T) {
	type args struct {
		configPath string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "should load the configuration file without errors",
			args: args{
				configPath: "./../../test/testdata/config_valid.json",
			},
			wantErr: false,
		},
		{
			name: "should not load the configuration due to wrong path",
			args: args{
				configPath: "./../../test/testdata/invalid.json",
			},
			wantErr: true,
		},
		{
			name: "should not load the configuration due to invalid port",
			args: args{
				configPath: "./../../test/testdata/config_invalid_port.json",
			},
			wantErr: true,
		},
		{
			name: "should not load the configuration due to invalid private file",
			args: args{
				configPath: "./../../test/testdata/config_invalid_private_key.json",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(tt.args.configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
