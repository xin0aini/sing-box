package route

import "github.com/sagernet/sing-box/adapter"

func (r *Router) ListProxyProvider() []adapter.ProxyProvider {
	return r.proxyProviders
}

func (r *Router) GetProxyProvider(tag string) adapter.ProxyProvider {
	if r.proxyProviderByTag != nil {
		return r.proxyProviderByTag[tag]
	}
	return nil
}

func (r *Router) ListProxyProviderOutbounds() map[string][]adapter.Outbound {
	return r.proxyProviderOutbounds
}

func (r *Router) GetProxyProviderOutbound(tag string) []adapter.Outbound {
	if r.proxyProviderOutbounds != nil {
		return r.proxyProviderOutbounds[tag]
	}
	return nil
}
