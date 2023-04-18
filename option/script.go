package option

import (
	"encoding/json"
	E "github.com/sagernet/sing/common/exceptions"
)

type ScriptOptions struct {
	Tag        string           `json:"tag,omitempty"`
	Mode       string           `json:"mode,omitempty"`
	Script     Listable[string] `json:"script,omitempty"`
	Keep       bool             `json:"keep,omitempty"`
	Timeout    Duration         `json:"timeout"`
	Output     bool             `json:"output"`
	IgnoreFail bool             `json:"ignore_fail"`
}

type _scriptOptions ScriptOptions

func (s *ScriptOptions) UnmarshalJSON(content []byte) error {
	err := json.Unmarshal(content, (*_scriptOptions)(s))
	if err != nil {
		return err
	}
	switch s.Mode {
	case "start-pre":
	case "start-post":
	case "close-pre":
	case "close-post":
	default:
		return E.New("script: invalid mode: ", s.Mode)
	}
	if s.Script == nil || len(s.Script) == 0 {
		return E.New("script is null")
	}
	return nil
}
