// Package proxy implements an HTTP/HTTPS proxy server that intercepts
// WeChat Channels (视频号) video requests to extract download URLs.
package proxy

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"

	"github.com/elazarl/goproxy"
)

// VideoInfo holds metadata about a captured video.
type VideoInfo struct {
	URL      string
	FileID   string
	Headers  http.Header
}

// Proxy wraps goproxy and maintains a list of captured video URLs.
type Proxy struct {
	server    *goproxy.ProxyHttpServer
	port      int
	mu        sync.Mutex
	videos    []VideoInfo
	OnCapture func(info VideoInfo)
}

// New creates a new Proxy instance listening on the given port.
func New(port int) *Proxy {
	p := &Proxy{
		port:   port,
		server: goproxy.NewProxyHttpServer(),
	}
	p.server.Verbose = false
	p.setupHandlers()
	return p
}

// setupHandlers registers request/response handlers to intercept video URLs.
func (p *Proxy) setupHandlers() {
	// Handle HTTPS CONNECT tunneling
	p.server.OnRequest().HandleConnect(goproxy.AlwaysMitm)

	// Intercept responses that look like WeChat Channels video segments
	p.server.OnResponse(goproxy.UrlMatches(videoURLPattern())).DoFunc(
		func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
			if resp == nil {
				return resp
			}
			rawURL := ctx.Req.URL.String()
			if isVideoURL(rawURL) {
				info := VideoInfo{
					URL:     rawURL,
					FileID:  extractFileID(rawURL),
					Headers: ctx.Req.Header.Clone(),
				}
				p.mu.Lock()
				p.videos = append(p.videos, info)
				p.mu.Unlock()
				log.Printf("[proxy] captured video URL: %s", rawURL)
				if p.OnCapture != nil {
					p.OnCapture(info)
				}
			}
			return resp
		},
	)
}

// Start begins listening for proxy connections.
func (p *Proxy) Start() error {
	addr := fmt.Sprintf(":%d", p.port)
	log.Printf("[proxy] starting on %s", addr)

	srv := &http.Server{
		Addr:    addr,
		Handler: p.server,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // required for MITM proxy
		},
	}
	return srv.ListenAndServe()
}

// Videos returns a snapshot of all captured video infos.
func (p *Proxy) Videos() []VideoInfo {
	p.mu.Lock()
	defer p.mu.Unlock()
	result := make([]VideoInfo, len(p.videos))
	copy(result, p.videos)
	return result
}

// ClearVideos removes all captured video entries.
func (p *Proxy) ClearVideos() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.videos = nil
}

// DumpRequest returns a human-readable dump of an HTTP request (for debugging).
func DumpRequest(req *http.Request) string {
	b, err := httputil.DumpRequest(req, false)
	if err != nil {
		return fmt.Sprintf("error dumping request: %v", err)
	}
	return string(b)
}

// videoURLPattern returns a pattern matching WeChat Channels video CDN hosts.
func videoURLPattern() interface{ MatchString(string) bool } {
	// Compiled lazily via strings match in isVideoURL; return a permissive regex.
	return regexpAny{}
}

// isVideoURL returns true if the URL looks like a WeChat Channels video resource.
func isVideoURL(rawURL string) bool {
	cdnHosts := []string{
		"finder.video.qq.com",
		"channels.weixin.qq.com",
		"vweixinf.tc.qq.com",
		"vweixin.tc.qq.com",
	}
	for _, host := range cdnHosts {
		if strings.Contains(rawURL, host) {
			return true
		}
	}
	return false
}

// extractFileID attempts to pull a meaningful file identifier from the URL.
func extractFileID(rawURL string) string {
	parts := strings.Split(rawURL, "/")
	if len(parts) == 0 {
		return ""
	}
	last := parts[len(parts)-1]
	// Strip query string
	if idx := strings.Index(last, "?"); idx != -1 {
		last = last[:idx]
	}
	return last
}

// regexpAny is a trivial matcher that matches every string (used as a
// placeholder so goproxy registers the handler for all responses; actual
// filtering is done in isVideoURL).
type regexpAny struct{}

func (regexpAny) MatchString(_ string) bool { return true }
