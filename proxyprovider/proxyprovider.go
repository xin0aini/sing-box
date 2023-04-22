//go:build with_proxyprovider

package proxyprovider

import (
	"context"
	"github.com/sagernet/sing-box/adapter"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-box/outbound"
	E "github.com/sagernet/sing/common/exceptions"
	F "github.com/sagernet/sing/common/format"
	"time"
)

func NewProxyProvider(ctx context.Context, router adapter.Router, logFactory log.Factory, options option.ProxyProviderOptions) (*ProxyProvider, error) {
	if options.Tag == "" {
		return nil, E.New("tag is required")
	}
	if options.URL == "" {
		return nil, E.New("url is required")
	}

	p := &ProxyProvider{
		tag:        options.Tag,
		ctx:        ctx,
		router:     router,
		logFactory: logFactory,
		options:    options,
	}

	p.initRequestDialer()
	err := p.initDNS()
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (p *ProxyProvider) Tag() string {
	return p.tag
}

func (p *ProxyProvider) GetOutboundOptions() ([]option.Outbound, error) {
	if p.peerList == nil {
		return nil, E.New("peer list is empty")
	}

	outbounds := make([]option.Outbound, 0)
	for i, px := range p.peerList {
		outboundOptions, err := px.GenerateOptions()
		if err != nil {
			continue
		}
		tag := px.Tag()
		if tag == "" {
			tag = F.ToString(p.Tag(), "-", i)
		}
		outbounds = append(outbounds, *outboundOptions)
	}

	groupOutbounds := p.getCustomGroupOptions(&outbounds)
	if groupOutbounds != nil {
		outbounds = append(outbounds, groupOutbounds...)
	}

	globalGroupTags := make([]string, 0)
	for _, out := range outbounds {
		globalGroupTags = append(globalGroupTags, out.Tag)
	}

	globalOutboundOptions := option.Outbound{}
	globalOutboundOptions.Tag = p.Tag()
	globalOutboundOptions.Type = C.TypeSelector
	globalOutboundOptions.SelectorOptions = option.SelectorOutboundOptions{
		Outbounds: globalGroupTags,
	}
	if p.options.DefaultOutbound != "" {
		for _, t := range globalGroupTags {
			if t == p.options.DefaultOutbound {
				globalOutboundOptions.SelectorOptions.Default = t
				break
			}
		}
	}

	outbounds = append(outbounds, globalOutboundOptions)

	return outbounds, nil
}

func (p *ProxyProvider) GetOutbounds() ([]adapter.Outbound, error) {
	if p.peerList == nil {
		return nil, E.New("peer list is empty")
	}

	outbounds := make([]adapter.Outbound, 0)
	for i, px := range p.peerList {
		outboundOptions, err := px.GenerateOptions()
		if err != nil {
			continue
		}
		tag := px.Tag()
		if tag == "" {
			tag = F.ToString(p.Tag(), "-", i)
		}
		out, err := outbound.New(p.ctx, p.router, p.logFactory.NewLogger(F.ToString("outbound/", outboundOptions.Type, "[", tag, "]")), tag, *outboundOptions)
		if err != nil {
			return nil, E.Cause(err, "parse proxyprovider[", p.Tag(), "] outbound[", tag, "]")
		}
		outbounds = append(outbounds, out)
	}

	groupOutbounds := p.getCustomGroups(outbounds)
	if groupOutbounds != nil {
		outbounds = append(outbounds, groupOutbounds...)
	}

	globalGroupTags := make([]string, 0)
	for _, out := range outbounds {
		globalGroupTags = append(globalGroupTags, out.Tag())
	}

	globalOutboundOptions := option.Outbound{}
	globalOutboundOptions.Tag = p.Tag()
	globalOutboundOptions.Type = C.TypeSelector
	globalOutboundOptions.SelectorOptions = option.SelectorOutboundOptions{
		Outbounds: globalGroupTags,
	}

	globalOut, err := outbound.New(p.ctx, p.router, p.logFactory.NewLogger(F.ToString("outbound/", globalOutboundOptions.Type, "[", globalOutboundOptions.Tag, "]")), globalOutboundOptions.Tag, globalOutboundOptions)
	if err != nil {
		return nil, E.Cause(err, "parse proxyprovider[", p.Tag(), "] outbound[", globalOutboundOptions.Tag, "]")
	}

	outbounds = append(outbounds, globalOut)

	return outbounds, nil
}

func (p *ProxyProvider) getCustomGroups(outbounds []adapter.Outbound) []adapter.Outbound {
	if p.options.CustomGroup == nil || len(p.options.CustomGroup) == 0 {
		return nil
	}

	group := make([]adapter.Outbound, 0)
	for i, g := range p.options.CustomGroup {
		if g.Tag == "" {
			g.Tag = F.ToString(p.Tag(), "-", i)
		}
		outs := make([]string, 0)
		for _, out := range outbounds {
			if CheckFilter(&g.ProxyProviderFilterOptions, out.Tag(), out.Type()) {
				outs = append(outs, out.Tag())
			}
		}
		if len(outs) == 0 {
			continue
		}
		groupOutOptions := option.Outbound{}
		switch g.Type {
		case C.TypeSelector:
			groupOutOptions.Tag = g.Tag
			groupOutOptions.Type = C.TypeSelector
			groupOutOptions.SelectorOptions = g.SelectorOptions
			groupOutOptions.SelectorOptions.Outbounds = outs
		case C.TypeURLTest:
			groupOutOptions.Tag = g.Tag
			groupOutOptions.Type = C.TypeURLTest
			groupOutOptions.URLTestOptions = g.URLTestOptions
			groupOutOptions.URLTestOptions.Outbounds = outs
		default:
			continue
		}
		groupOut, err := outbound.New(p.ctx, p.router, p.logFactory.NewLogger(F.ToString("outbound/", groupOutOptions.Type, "[", groupOutOptions.Tag, "]")), groupOutOptions.Tag, groupOutOptions)
		if err != nil {
			continue
		}
		group = append(group, groupOut)
	}

	if len(group) == 0 {
		return nil
	}

	return group
}

func (p *ProxyProvider) getCustomGroupOptions(outbounds *[]option.Outbound) []option.Outbound {
	if p.options.CustomGroup == nil || len(p.options.CustomGroup) == 0 {
		return nil
	}

	group := make([]option.Outbound, 0)
	for i, g := range p.options.CustomGroup {
		if g.Tag == "" {
			g.Tag = F.ToString(p.Tag(), "-", i)
		}
		outs := make([]string, 0)
		for _, out := range *outbounds {
			if CheckFilter(&g.ProxyProviderFilterOptions, out.Tag, out.Type) {
				outs = append(outs, out.Tag)
			}
		}
		if len(outs) == 0 {
			continue
		}
		groupOutOptions := option.Outbound{}
		switch g.Type {
		case C.TypeSelector:
			groupOutOptions.Tag = g.Tag
			groupOutOptions.Type = C.TypeSelector
			groupOutOptions.SelectorOptions = g.SelectorOptions
			groupOutOptions.SelectorOptions.Outbounds = outs
		case C.TypeURLTest:
			groupOutOptions.Tag = g.Tag
			groupOutOptions.Type = C.TypeURLTest
			groupOutOptions.URLTestOptions = g.URLTestOptions
			groupOutOptions.URLTestOptions.Outbounds = outs
		default:
			continue
		}
		group = append(group, groupOutOptions)
	}

	if len(group) == 0 {
		return nil
	}

	return group
}

func (p *ProxyProvider) GetUpdateTime() time.Time {
	p.subscriptionRawDataLock.RLock()
	defer p.subscriptionRawDataLock.RUnlock()
	return p.subscriptionRawData.UpdateTime
}

func (p *ProxyProvider) GetSubscribeInfo() adapter.SubScribeInfo {
	p.subscriptionRawDataLock.RLock()
	defer p.subscriptionRawDataLock.RUnlock()
	return &p.subscriptionRawData.SubScribeInfo
}
