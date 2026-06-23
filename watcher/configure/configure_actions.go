package configure

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"

	"gopkg.in/yaml.v3"
)

// var _ yaml.Unmarshaler = (*MonitoredObjectAction)(nil)

type MonitoredObject struct {
	Recursive bool                    `yaml:"recursive" json:"recursive"`
	Actions   []MonitoredObjectAction `yaml:"actions" json:"actions"`
}

type MonitoredObjectAction struct {
	ID         string     `yaml:"id" json:"id"`
	ActionType ActionType `yaml:"action_type" json:"action_type"` // maxsize | create | delete | rename | change
	Size       uint64     `yaml:"size" json:"size"`               // max size in bytes for action
	Debounce   float64    `yaml:"debounce" json:"debounce"`       // time to wait in seconds for debouncing, min 0.1
	Action     string     `yaml:"action" json:"action"`           // path to shell or javascript file
	Cron       string     `yaml:"cronjob" json:"cronjob"`         // schedule for cronjob, ref github.com/robfig/cron/v3
	// Supervised bool       `yaml:"supervised" json:"supervised"`   // user can decide and see which commands / actions are ran, interactive.
}

func getActionDir(monitorPath string) (dir string, err error) {
	dir, err = Dir()
	if err != nil {
		return "", err
	}

	var hash = md5.New()
	hash.Write([]byte(monitorPath))

	return filepath.Join(
		dir,
		"actions",
		fmt.Sprintf("%x", hash.Sum(nil)),
	), nil
}

func getActionName(actionPath string) (name string, err error) {
	if actionPath == "" { // TODO: this is a hack to get the action name from the action path in the config file. We should fix it better.
		return "", errors.New("No Action Path specified")
	}

	var suffix = filepath.Ext(actionPath)
	if suffix == "" {
		return "", fmt.Errorf("action %q missing suffix", actionPath)
	}

	var hash = md5.New()
	hash.Write([]byte(actionPath))
	return fmt.Sprintf("%x%s", hash.Sum(nil), suffix), nil
}

func SaveActionFile(monitorPath string, srcActionPath string) (newActionPath string, err error) {
	writeTo, err := getActionDir(monitorPath)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(writeTo, os.ModeDir|os.ModeExclusive); err != nil && !errors.Is(err, os.ErrExist) {
		return "", fmt.Errorf("Failed to create action directory %q: %w", writeTo, err)
	}

	actionName, err := getActionName(srcActionPath)
	if err != nil {
		return "", err
	}

	srcFile, err := os.OpenFile(srcActionPath, os.O_RDONLY, 0o755)
	if err != nil {
		return "", fmt.Errorf("Failed to read action source file %q: %w", srcActionPath, err)
	}
	defer srcFile.Close()

	newActionPath = filepath.Join(writeTo, actionName)
	dstFile, err := os.OpenFile(newActionPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return "", fmt.Errorf("Failed to open action destination file %q: %w", srcActionPath, err)
	}
	defer dstFile.Close()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return "", fmt.Errorf("Failed to copy action file %q to %q: %w", srcActionPath, newActionPath, err)
	}

	return newActionPath, nil
}

func DeleteActionFile(action MonitoredObjectAction) error {
	return os.Remove(action.Action)
}

func (m *MonitoredObjectAction) UnmarshalYAML(node *yaml.Node) error {
	var data = make(map[string]any)
	if err := node.Decode(&data); err != nil {
		return err
	}

	var rv = reflect.ValueOf(m).Elem()
	for f, fVal := range rv.Fields() {

		dVal, ok := data[f.Tag.Get("yaml")]
		if !ok {
			continue
		}

		rdVal := reflect.ValueOf(dVal)
		rdTyp := rdVal.Type()

		if rdTyp.AssignableTo(f.Type) {
			fVal.Set(rdVal)
			continue
		}

		if !rdVal.Type().ConvertibleTo(f.Type) {
			return errors.New("field type mismatch")
		}

		rdVal = rdVal.Convert(f.Type)
		fVal.Set(rdVal)
	}

	if m.ID == "" {
		return errors.New("Action has no ID")
	}

	// debounce is checked / validated in watcher

	return nil
}

func (m MonitoredObjectAction) MarshalYAML() (interface{}, error) {
	var n = &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
	}

	var nodes = []*yaml.Node{
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "id"},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: m.ID},

		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "action_type"},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: m.ActionType},

		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "size"},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.FormatUint(m.Size, 10)},

		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "debounce"},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!float", Value: strconv.FormatFloat(m.Debounce, 'g', -1, 64)},

		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "action"},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: m.Action},

		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "cronjob"},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: m.Cron},
	}

	n.Content = nodes

	return n, nil
}
