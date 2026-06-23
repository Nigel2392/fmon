package configure

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "embed"

	"github.com/Nigel2392/go-django/pkg/yml"
	"github.com/elliotchance/orderedmap/v2"
	"gopkg.in/yaml.v3"
)

//go:embed header.comment
var yaml_header []byte

func IsUserConfig(path string) bool {
	for _, v := range SETUP.BuiltUserPaths() {
		if strings.EqualFold(path, os.ExpandEnv(v)) {
			return true
		}
	}
	return false
}

func ExistingPath() (string, error) {
	for _, path := range SETUP.AllLocations() {
		path = os.ExpandEnv(path)
		_, err := os.Stat(path)
		if err != nil && os.IsNotExist(err) {
			continue
		} else if err != nil {
			return "", err
		}
		return path, nil

	}

	return "", ErrConfigNotExists
}

func Dir() (string, error) {
	var cnfPath, err = ExistingPath()
	if err != nil {
		return "", err
	}
	return filepath.Dir(cnfPath), nil
}

func Read() (*FilesystemMonitor, error) {
	for _, cnfPath := range SETUP.AllLocations() {
		cnfPath := os.ExpandEnv(cnfPath)
		if _, err := os.Stat(cnfPath); err != nil {
			continue
		}

		cnfExt := cnfPath[strings.LastIndex(cnfPath, ".")+1:]
		unmarshal, ok := SETUP.Unmarshallers[cnfExt]
		if !ok {
			panic("This really shouldn't happen.")
		}

		data, err := os.ReadFile(cnfPath)
		if err != nil {
			return nil, fmt.Errorf("Failed to read file %s: %w", cnfPath, err)
		}

		var configFile = NewMonitorConfig(cnfExt, cnfPath)
		if err := unmarshal(data, configFile); err != nil {
			return nil, fmt.Errorf("Failed to parse file %s: %w", cnfPath, err)
		}

		return configFile, nil
	}

	return nil, ErrConfigNotExists
}

func Rewrite(config *FilesystemMonitor, global bool) error {
	if config.Path != "" {
		return Write(config.Path, config)
	}

	var locations = SETUP.BuiltUserPaths()
	if global {
		locations = SETUP.BuiltGlobalPaths()
	}

	var written bool
	var errs = make([]error, 0)
	for _, path := range locations {
		path := os.ExpandEnv(path)
		if _, err := os.Stat(path); err != nil && !os.IsNotExist(err) {
			continue
		}

		config.Path = path
		if err := Write(path, config); err != nil {
			errs = append(errs, err)
			continue
		}

		written = true
		break
	}

	if written {
		return nil
	}

	return errors.Join(errs...)
}

func Write(to string, config *FilesystemMonitor) error {
	if to == "" {
		panic("Please specify a path to write the config file to.")
	}

	if config.Type == "" {
		config.Type = DEFAULT_CONFIG_TYPE
	}

	marshal, ok := SETUP.Marshallers[config.Type]
	if !ok {
		panic(fmt.Sprintf("This really shouldn't happen. Type %q not found in %v", config.Type, SETUP.Marshallers))
	}

	data, err := marshal(config)
	if err != nil {
		return fmt.Errorf("Failed to write config file %s: %w", config.Path, err)
	}

	var dirBase = filepath.Dir(to)
	var dirName = filepath.Base(dirBase)
	if dirName == CONFIG_NAME_BASE {
		if err := os.MkdirAll(dirBase, 0755); err != nil && !os.IsExist(err) {
			return fmt.Errorf("Failed to create directory %s: %w", dirBase, err)
		}
	}

	f, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("Failed to open file %s for writing: %w", to, err)
	}

	_, err = f.Write(yaml_header)
	if err != nil {
		return fmt.Errorf("Failed to write file %s: %w", to, err)
	}

	_, err = f.Write(data)
	if err != nil {
		return fmt.Errorf("Failed to write file %s: %w", to, err)
	}

	return nil
}

type ActionType = string

const (
	TICKER_ACTION ActionType = "ticker"
	CREATE_ACTION ActionType = "create"
	DELETE_ACTION ActionType = "delete"
	RENAME_ACTION ActionType = "rename"
	CHANGE_ACTION ActionType = "change"
)

var ACTION_TYPES = []ActionType{
	TICKER_ACTION,
	CREATE_ACTION,
	DELETE_ACTION,
	RENAME_ACTION,
	CHANGE_ACTION,
}

type FilesystemMonitor struct {
	Type  string                                    `yaml:"-" json:"-"`
	Path  string                                    `yaml:"-" json:"-"`
	Files *yml.OrderedMap[string, *MonitoredObject] `yaml:",inline" json:"files"`
}

func NewMonitorConfig(typ, path string) *FilesystemMonitor {
	return &FilesystemMonitor{
		Type: typ,
		Path: path,
		Files: &yml.OrderedMap[string, *MonitoredObject]{
			OrderedMap: orderedmap.NewOrderedMap[string, *MonitoredObject](),
		},
	}
}

func (f *FilesystemMonitor) MarshalYAML() (interface{}, error) {
	var root = &yaml.Node{
		Kind:    yaml.MappingNode,
		Tag:     "!!map",
		Content: make([]*yaml.Node, f.Files.Len()*2),
	}

	var idx = 0
	for head := f.Files.Front(); head != nil; head = head.Next() {
		var (
			keyNode = new(yaml.Node)
			valNode = new(yaml.Node)
		)

		if err := keyNode.Encode(head.Key); err != nil {
			return nil, fmt.Errorf("error marshaling key %v: %w", head.Key, err)
		}

		if err := valNode.Encode(head.Value); err != nil {
			return nil, fmt.Errorf("error marshaling key %v: %w", head.Key, err)
		}

		root.Content[idx] = keyNode
		root.Content[idx+1] = valNode

		idx += 2
	}

	return root, nil
}
