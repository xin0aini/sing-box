//go:build with_proxyprovider

package proxyprovider

import (
	"context"
	"github.com/sagernet/sing-box/proxyprovider/proxy"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	"gopkg.in/yaml.v3"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func (p *ProxyProvider) update() error {
	cache, cacheTime, cacheErr := p.readCache()
	if cacheErr == nil {
		if p.options.ForceUpdate == 0 || time.Since(cacheTime) < time.Duration(p.options.ForceUpdate) {
			p.subscriptionRawData = *cache
			return nil
		}
	}

	rawData, err := p.request()
	if err != nil {
		if cacheErr == nil {
			return nil
		}
		return E.Cause(err, "failed to update proxy provider")
	}

	p.subscriptionRawData = *rawData

	err = p.writeCache()
	if err != nil {
		return err
	}

	return nil
}

func (p *ProxyProvider) Update() error {
	if !p.updateLock.TryLock() {
		return nil
	}
	defer p.updateLock.Unlock()

	err := p.update()
	if err != nil {
		return err
	}

	return p.parseToPeerList()
}

func (p *ProxyProvider) ForceUpdate() error {
	if !p.updateLock.TryLock() {
		return nil
	}
	defer p.updateLock.Unlock()

	rawData, err := p.request()
	if err != nil {
		return E.Cause(err, "failed to update proxy provider")
	}

	p.subscriptionRawData = *rawData

	err = p.writeCache()
	if err != nil {
		return err
	}

	return p.parseToPeerList()
}

func (p *ProxyProvider) parseToPeerList() error {
	var clashConfig proxy.ClashConfig
	err := yaml.Unmarshal(p.subscriptionRawData.PeerInfo, &clashConfig)
	if err != nil {
		return p.parseToPeerListFormLink()
	}
	if clashConfig.Proxies == nil || len(clashConfig.Proxies) == 0 {
		return E.New("no proxies found")
	}

	proxies := make([]proxy.Proxy, 0)
	for _, proxyConfig := range clashConfig.Proxies {
		px, err := proxyConfig.ToProxy()
		if err != nil {
			return E.Cause(err, "failed to parse proxy")
		}
		if !CheckFilter(p.options.Filter, px.Tag(), px.Type()) {
			continue
		}
		if p.options.DialerOptions != nil {
			px.SetDialerOptions(*p.options.DialerOptions)
		}
		proxies = append(proxies, px)
	}

	p.peerList = proxies

	return nil
}

func (p *ProxyProvider) parseToPeerListFormLink() error {
	return E.New("failed to parse peer info")
}

func (p *ProxyProvider) readCache() (*subscriptionRawData, time.Time, error) {
	file, err := os.Open(p.options.CacheFile)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, time.Time{}, err
	}

	data := make([]byte, fileInfo.Size())

	var n int

	n, err = file.Read(data)
	if err != nil {
		return nil, time.Time{}, err
	}

	if n == 0 {
		return nil, time.Time{}, E.New("empty cache file")
	}

	var s subscriptionRawData

	err = s.decode(data)
	if err != nil {
		return nil, time.Time{}, err
	}

	return &s, fileInfo.ModTime(), nil
}

func (p *ProxyProvider) writeCache() error {
	if p.options.CacheFile != "" {
		data, err := p.subscriptionRawData.encode()
		if err != nil {
			return err
		}

		return os.WriteFile(p.options.CacheFile, data, 0o644)
	}

	return nil
}

func (p *ProxyProvider) request() (*subscriptionRawData, error) {
	u, err := url.Parse(p.options.URL)
	if err != nil {
		return nil, E.Cause(err, "failed to parse url")
	}

	port := u.Port()
	if port == "" {
		if u.Scheme == "http" {
			port = "80"
		} else if u.Scheme == "https" {
			port = "443"
		}
	}

	addr := u.Hostname()
	ip, err := netip.ParseAddr(addr)
	if err != nil {
		ips, err := p.query(addr)
		if err != nil {
			return nil, E.Cause(err, "failed to resolve domain")
		}
		ip = ips[0]
	}

	remoteAddr := net.JoinHostPort(ip.String(), port)

	req, err := http.NewRequest(http.MethodGet, p.options.URL, nil)
	if err != nil {
		return nil, E.Cause(err, "failed to create request")
	}
	req.Header.Set("User-Agent", "clash")
	req.RemoteAddr = remoteAddr

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return p.dialer.DialContext(ctx, network, M.ParseSocksaddr(addr))
			},
			ForceAttemptHTTP2: true,
		},
	}

	reqCtx, reqCancel := context.WithTimeout(p.ctx, time.Second*30)
	defer reqCancel()

	req = req.WithContext(reqCtx)

	resp, err := client.Do(req)
	if err != nil {
		return nil, E.Cause(err, "failed to request")
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, E.Cause(err, "failed to read response")
	}

	s := &subscriptionRawData{
		PeerInfo: data,
	}
	s.UpdateTime = time.Now()

	subscriptionInfo := resp.Header.Get("subscription-userinfo")
	if subscriptionInfo != "" {
		subscriptionInfo = strings.ToLower(subscriptionInfo)
		regTraffic := regexp.MustCompile("upload=(\\d+); download=(\\d+); total=(\\d+)")
		matchTraffic := regTraffic.FindStringSubmatch(subscriptionInfo)
		if len(matchTraffic) == 4 {
			uploadUint64, err := strconv.ParseUint(matchTraffic[1], 10, 64)
			if err == nil {
				s.Upload = uploadUint64
			}
			downloadUint64, err := strconv.ParseUint(matchTraffic[2], 10, 64)
			if err == nil {
				s.Download = downloadUint64
			}
			totalUint64, err := strconv.ParseUint(matchTraffic[3], 10, 64)
			if err == nil {
				s.Total = totalUint64
			}
		}
		regExpire := regexp.MustCompile("expire=(\\d+)")
		matchExpire := regExpire.FindStringSubmatch(subscriptionInfo)
		if len(matchExpire) == 2 {
			expireUint64, err := strconv.ParseUint(matchExpire[1], 10, 64)
			if err == nil {
				s.Expire = time.Unix(int64(expireUint64), 0)
			}
		}
	}

	return s, nil
}
