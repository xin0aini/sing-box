# sing-box

The universal proxy platform.

[![Packaging status](https://repology.org/badge/vertical-allrepos/sing-box.svg)](https://repology.org/project/sing-box/versions)

## Documentation

https://sing-box.sagernet.org

## Support

https://community.sagernet.org/c/sing-box/

## License

```
Copyright (C) 2022 by nekohasekai <contact-sagernet@sekai.icu>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.

In addition, no derivative work may use the name or imply association
with this application without prior consent.
```

## 额外功能

```
1. Proxy-Provider 支持 (with_proxyprovider)

{
    "proxyproviders": [ // proxy-provider 配置，参考下方，若只有一项，可省略[]
        {
            "tag": "proxy-provider-1", // 标签，必填，用于区别不同的 proxy-provider
            "url": "https://www.google.com", // 订阅链接，必填，仅支持Clash订阅链接
            "cache_file": "/tmp/proxy-provider-1.cache", // 缓存文件，选填，强烈建议填写，可以加快启动速度
            "force_update": "4h", // 强制更新间隔，选填，若当前缓存文件已经超过该时间，将会强制更新
            "dns": "tcp://223.5.5.5", // 请求的DNS服务器，选填，若不填写，将会选择默认DNS
            "filter": { // 过滤节点，选填
                "rule": [
                    "到期"
                ], // 过滤规则，选填，若只有一项，可省略[]
                "white_mode": false // 白名单模式（只保留匹配的节点），选填，若不填写，将会使用黑名单模式（只保留未匹配的节点）
            },
            "request_dialer": {}, // 请求的Dialer，选填，详见sing-box dialer字段，不支持detour, domain_strategy, fallback_delay
            "dialer": {}, // 节点的Dialer，选填，详见sing-box dialer字段
            "custom_group": [ // 自定义分组，选填，若只有一项，可省略[]
                {
                    "tag": "selector-1", // outbound tag，必填
                    "type": "selector", // outbound 类型，必填，仅支持selector, urltest
                    "rule": [], // 节点过滤规则，选填，详见上filter.rule字段
                    "white_mode": false, // 节点过滤模式，选填，详见上filter.white_mode字段
                    ... // selector或urltest的其他字段，选填
                },
                ...
            ]
        }
    ],
    "outbounds": [...
    ]
}


```