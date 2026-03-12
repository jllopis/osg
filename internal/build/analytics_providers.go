package build

import (
	"fmt"
	"html"
	"strings"

	"osg/internal/config"
)

// analyticsHeadSnippets returns the combined HTML to inject at the end of <head>
// for all configured third-party analytics providers.
func analyticsHeadSnippets(providers []config.AnalyticsProviderConfig) string {
	if len(providers) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, p := range providers {
		switch p.Provider {
		case "cloudflare":
			fmt.Fprintf(&sb, `<script defer src="https://static.cloudflareinsights.com/beacon.min.js" data-cf-beacon='{"token":"%s"}'></script>`+"\n", html.EscapeString(p.Token))
		case "google":
			id := html.EscapeString(p.TrackingID)
			fmt.Fprintf(&sb, `<script async src="https://www.googletagmanager.com/gtag/js?id=%s"></script>`+"\n", id)
			fmt.Fprintf(&sb, `<script>window.dataLayer=window.dataLayer||[];function gtag(){dataLayer.push(arguments);}gtag('js',new Date());gtag('config','%s');</script>`+"\n", id)
		case "plausible":
			fmt.Fprintf(&sb, `<script defer data-domain="%s" src="https://plausible.io/js/script.js"></script>`+"\n", html.EscapeString(p.Domain))
		case "fathom":
			fmt.Fprintf(&sb, `<script src="https://cdn.usefathom.com/script.js" data-site="%s" defer></script>`+"\n", html.EscapeString(p.Token))
		}
	}
	return sb.String()
}
