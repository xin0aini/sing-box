//go:build with_proxyprovider

package proxy

import (
	"runtime/debug"
	"strings"
)

var tagMap map[string]bool

func init() {
	debugInfo, loaded := debug.ReadBuildInfo()
	if loaded {
		for _, setting := range debugInfo.Settings {
			switch setting.Key {
			case "-tags":
				tags := setting.Value
				tagMap = make(map[string]bool)
				for _, tag := range strings.Split(tags, ",") {
					tagMap[tag] = true
				}
			}
		}
	}
}

func GetTag(tag string) bool {
	if tagMap == nil {
		return false
	}
	return tagMap[tag]
}
