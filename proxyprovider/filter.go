//go:build with_proxyprovider

package proxyprovider

import "github.com/sagernet/sing-box/option"

func CheckFilter(f *option.ProxyProviderFilterOptions, tag string, _type string) bool { // save: true
	if f != nil && f.Rule != nil && len(f.Rule) > 0 {
		match := false

		for _, rule := range f.Rule {
			if rule.Match(tag, _type) {
				match = true
				break
			}
		}

		if f.WhiteMode {
			return match
		} else {
			return !match
		}
	}

	return true
}
