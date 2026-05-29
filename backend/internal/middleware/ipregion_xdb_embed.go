//go:build embed_ipregion

package middleware

import _ "embed"

//go:embed data/ip2region_v4.xdb
var embeddedIPv4XdbData []byte

//go:embed data/ip2region_v6.xdb
var embeddedIPv6XdbData []byte
