package setflag

import (
	"fmt"
	"strings"
)

func New(options ...string) *SetFlag {
	sf := &SetFlag{
		values:  make(map[string]struct{}, len(options)),
		options: make(map[string]struct{}, len(options)),
	}
	for _, opt := range options {
		sf.options[opt] = struct{}{}
	}
	return sf
}

type SetFlag struct {
	options map[string]struct{}
	values  map[string]struct{}
}

func (sf *SetFlag) List() []string {
	var values []string
	for k := range sf.values {
		values = append(values, k)
	}
	return values
}

func (sf *SetFlag) String() string {
	return strings.Join(sf.List(), ", ")
}

func (sf *SetFlag) Set(value string) error {
	values := []string{value}
	if strings.Contains(value, ",") {
		values = strings.Split(value, ",")
		for i, str := range values {
			values[i] = strings.TrimSpace(str)
		}
	}
	for _, value := range values {
		if _, exists := sf.options[value]; !exists {
			return fmt.Errorf("unsupported value '%s'", value)
		}
		sf.values[value] = struct{}{}
	}
	return nil
}
