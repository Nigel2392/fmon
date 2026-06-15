package configure

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	DEFAULT_CONFIG_TYPE = "yaml"
	CONFIG_NAME_BASE    = "fmon"
)

var (
	SETUP      PackageSetup
	USER_PATHS = []string{
		"$UserProfile\\.%s\\%s.yaml",
		"$AppData\\%s\\%s.yaml",
		"~/.config/%s/%s.yaml",
	}
	GLOBAL_PATHS = []string{
		"$ProgramData\\%s\\%s.yaml",
		"/etc/%s/%s.yaml",
	}
	DEFAULT_MARSHALLERS = map[string]func(obj any) ([]byte, error){
		"yaml": func(obj any) ([]byte, error) { return yaml.Marshal(obj) },
	}
	DEFAULT_UNMARSHALLERS = map[string]func(in []byte, obj any) error{
		"yaml": func(in []byte, obj any) error { return yaml.Unmarshal(in, obj) },
	}
)

func Setup(p PackageSetup) {
	var marshallers = make(map[string]func(obj any) ([]byte, error))
	var unmarshallers = make(map[string]func(in []byte, obj any) error)

	maps.Copy(marshallers, DEFAULT_MARSHALLERS)
	maps.Copy(marshallers, p.Marshallers)

	maps.Copy(unmarshallers, DEFAULT_UNMARSHALLERS)
	maps.Copy(unmarshallers, p.Unmarshallers)

	if len(marshallers) == 0 || len(unmarshallers) == 0 {
		panic("No supported file types found.")
	}

	if p.DefaultType == "" {
		var keys = slices.Collect(maps.Keys(marshallers))
		slices.Sort(keys)
		p.DefaultType = keys[0]
	}

	if p.NameBase == "" {
		panic("Please specify a name base for the configuration file.")
	}

	p.Marshallers = marshallers
	p.Unmarshallers = unmarshallers

	SETUP = p
}

type PackageSetup struct {
	DefaultType   string
	NameBase      string
	Marshallers   map[string]func(obj any) ([]byte, error)
	Unmarshallers map[string]func(in []byte, obj any) error

	_builtUserLocations   []string
	_builtGlobalLocations []string
}

func (s *PackageSetup) BuiltUserPaths() []string {
	if s._builtUserLocations == nil {
		s._builtUserLocations = _buildDirPaths(s.NameBase, USER_PATHS)
	}
	return s._builtUserLocations
}

func (s *PackageSetup) BuiltGlobalPaths() []string {
	if s._builtGlobalLocations == nil {
		s._builtGlobalLocations = _buildDirPaths(s.NameBase, GLOBAL_PATHS)
	}
	return s._builtGlobalLocations
}

func (s *PackageSetup) AllLocations() []string {
	return append(
		s.BuiltUserPaths(),
		s.BuiltGlobalPaths()...,
	)
}

func _buildDirPaths(baseName string, paths []string) []string {
	for idx, path := range paths {
		var argCount = strings.Count(path, "%s")
		var args = make([]any, argCount)
		for i := range argCount {
			args[i] = baseName
		}
		paths[idx] = fmt.Sprintf(path, args...)
	}
	return paths
}
