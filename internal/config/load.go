package config

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadFile reads, strictly decodes, and validates a YAML configuration file.
func LoadFile(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()
	return Load(f)
}

// Load strictly decodes and validates YAML from r. Defaults are installed
// before decoding so omitted optional fields remain bounded and predictable.
func Load(r io.Reader) (Config, error) {
	const maxConfigBytes = 4 << 20
	data, err := io.ReadAll(io.LimitReader(r, maxConfigBytes+1))
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	if len(data) > maxConfigBytes {
		return Config{}, fmt.Errorf("read config: size exceeds %d bytes", maxConfigBytes)
	}
	cfg := Default()
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Config{}, fmt.Errorf("decode config: multiple YAML documents are not allowed")
		}
		return Config{}, fmt.Errorf("decode trailing config: %w", err)
	}
	if err := Validate(cfg); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}
	return cfg, nil
}
