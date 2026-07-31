// Copyright 2020 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import "testing"

func TestValidateLabelFlags(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		configFile           string
		label                string
		queryParam           string
		headerName           string
		labelValues          []string
		headerUsesListSyntax bool

		wantErr string
	}{
		{
			name:  "label only",
			label: "namespace",
		},
		{
			name:       "label and query parameter",
			label:      "namespace",
			queryParam: "namespace",
		},
		{
			name:       "label and header",
			label:      "namespace",
			headerName: "X-Namespace",
		},
		{
			name:        "multiple static values for one label",
			label:       "namespace",
			labelValues: []string{"team-a", "team-b"},
		},
		{
			name:       "config file only",
			configFile: "config.yaml",
		},
		{
			name:    "missing label",
			wantErr: "-label flag cannot be empty",
		},
		{
			name:       "mixed dynamic sources",
			label:      "namespace",
			queryParam: "namespace",
			headerName: "X-Namespace",
			wantErr:    "at most one of -query-param, -header-name and -label-value must be set",
		},
		{
			name:        "static and query parameter sources",
			label:       "namespace",
			queryParam:  "namespace",
			labelValues: []string{"team-a"},
			wantErr:     "at most one of -query-param, -header-name and -label-value must be set",
		},
		{
			name:        "static and header sources",
			label:       "namespace",
			headerName:  "X-Namespace",
			labelValues: []string{"team-a"},
			wantErr:     "at most one of -query-param, -header-name and -label-value must be set",
		},
		{
			name:       "config file and label",
			configFile: "config.yaml",
			label:      "namespace",
			wantErr:    "-config-file can't be combined with -label, -query-param, -header-name, -label-value or -header-uses-list-syntax",
		},
		{
			name:       "config file and query parameter",
			configFile: "config.yaml",
			queryParam: "namespace",
			wantErr:    "-config-file can't be combined with -label, -query-param, -header-name, -label-value or -header-uses-list-syntax",
		},
		{
			name:        "config file and static values",
			configFile:  "config.yaml",
			labelValues: []string{"team-a"},
			wantErr:     "-config-file can't be combined with -label, -query-param, -header-name, -label-value or -header-uses-list-syntax",
		},
		{
			name:                 "config file and header list syntax",
			configFile:           "config.yaml",
			headerUsesListSyntax: true,
			wantErr:              "-config-file can't be combined with -label, -query-param, -header-name, -label-value or -header-uses-list-syntax",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateLabelFlags(tc.configFile, tc.label, tc.queryParam, tc.headerName, tc.labelValues, tc.headerUsesListSyntax)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || err.Error() != tc.wantErr {
				t.Fatalf("expected error %q, got %v", tc.wantErr, err)
			}
		})
	}
}
