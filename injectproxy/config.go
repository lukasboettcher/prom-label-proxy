// Copyright 2026 The Prometheus Authors
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

package injectproxy

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/prometheus/common/model"
	"go.yaml.in/yaml/v3"
)

// Config declares the labels enforced by the proxy.
type Config struct {
	Labels []LabelConfig `yaml:"labels"`
}

// LabelConfig declares a single enforced label and the source of its values.
// Exactly one of Header, QueryParam or Values must be set.
type LabelConfig struct {
	Name       string        `yaml:"name"`
	Header     *HeaderConfig `yaml:"header,omitempty"`
	QueryParam string        `yaml:"query_param,omitempty"`
	Values     []string      `yaml:"values,omitempty"`
}

// HeaderConfig declares the HTTP header carrying the label values.
type HeaderConfig struct {
	Name string `yaml:"name"`
	// UsesListSyntax parses every header line as a comma-separated list of values.
	UsesListSyntax bool `yaml:"uses_list_syntax,omitempty"`
}

// LoadConfig reads and validates the configuration from the given file.
func LoadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("can't read the configuration file: %w", err)
	}

	cfg, err := ParseConfig(b)
	if err != nil {
		return nil, fmt.Errorf("invalid configuration file %q: %w", path, err)
	}

	return cfg, nil
}

// ParseConfig unmarshals and validates the YAML configuration.
func ParseConfig(b []byte) (*Config, error) {
	var cfg Config

	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, errors.New("the configuration is empty")
		}

		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate returns an error if the configuration isn't valid.
func (c Config) Validate() error {
	if len(c.Labels) == 0 {
		return errors.New("at least one label must be configured")
	}

	seen := make(map[string]struct{}, len(c.Labels))
	seenQueryParams := make(map[string]struct{}, len(c.Labels))
	for i, l := range c.Labels {
		if err := l.validate(); err != nil {
			return fmt.Errorf("labels[%d]: %w", i, err)
		}

		if _, found := seen[l.Name]; found {
			return fmt.Errorf("labels[%d]: label %q is configured more than once", i, l.Name)
		}
		seen[l.Name] = struct{}{}

		if l.QueryParam == "" {
			continue
		}

		// Sharing a query parameter between labels is rejected because the
		// extractors strip the parameter from the proxied request as they run.
		if _, found := seenQueryParams[l.QueryParam]; found {
			return fmt.Errorf("labels[%d]: query parameter %q is used by more than one label", i, l.QueryParam)
		}
		seenQueryParams[l.QueryParam] = struct{}{}
	}

	return nil
}

func (l LabelConfig) validate() error {
	if !model.UTF8Validation.IsValidLabelName(l.Name) {
		return fmt.Errorf("invalid label name %q", l.Name)
	}

	var sources []string
	if l.Header != nil {
		sources = append(sources, "header")
	}
	if l.QueryParam != "" {
		sources = append(sources, "query_param")
	}
	if len(l.Values) > 0 {
		sources = append(sources, "values")
	}

	if len(sources) != 1 {
		return fmt.Errorf("exactly one of header, query_param or values must be set for label %q, got %d", l.Name, len(sources))
	}

	if l.Header != nil && l.Header.Name == "" {
		return fmt.Errorf("the header name must be set for label %q", l.Name)
	}

	for _, v := range l.Values {
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("values must not be empty for label %q", l.Name)
		}
	}

	return nil
}

func (l LabelConfig) extractLabeler() ExtractLabeler {
	switch {
	case l.Header != nil:
		return HTTPHeaderEnforcer{
			Name:            http.CanonicalHeaderKey(l.Header.Name),
			ParseListSyntax: l.Header.UsesListSyntax,
		}
	case len(l.Values) > 0:
		return StaticLabelEnforcer(l.Values)
	default:
		return HTTPFormEnforcer{ParameterName: l.QueryParam}
	}
}

// WithConfig configures the proxy to enforce every label declared in cfg.
func WithConfig(cfg Config) Option {
	return optionFunc(func(o *options) {
		for _, l := range cfg.Labels {
			o.labels = append(o.labels, enforcedLabel{name: l.Name, extractLabeler: l.extractLabeler()})
		}
	})
}
