//go:build with_proxyprovider

package proxy

type proxyClashRealityOptions struct {
	PublicKey string `yaml:"public-key"`
	ShortID   string `yaml:"short-id"`
}
