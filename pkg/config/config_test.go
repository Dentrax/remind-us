/*
Copyright © 2021 Furkan Türkal

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"reflect"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    *Config
		wantErr bool
	}{
		{
			"it should not load if bad path",
			"./",
			nil,
			true,
		},
		{
			"it should not load if good path and json",
			"../../config.json",
			nil,
			true,
		},
		{
			"it should not load if good path and no ext",
			"../../config",
			nil,
			true,
		},
		{
			"it should load if good path and yaml",
			"../../config.yaml",
			&Config{
				IntegrationConfig{
					GitLab: GitLabIntegrationConfig{
						BaseURL: "https://gitlab.com",
						Token:   "xxx",
						Listen: IntegrationListenConfig{
							Areas: []IntegrationAreaConfig{
								{
									Type: "PR",
								},
							},
							Groups: []uint16{111, 222, 333},
						},
					},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Load(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Load() got = %v, want %v", got, tt.want)
			}
		})
	}
}
