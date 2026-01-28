package acme

import (
	"fmt"
	"net/http"
)

// HTTPProvider 實現 challenge.Provider 接口
type HTTPProvider struct {
	server *http.Server
}

func NewHTTPProvider() *HTTPProvider {
	return &HTTPProvider{}
}

// Present 啓動 HTTP 服務器
func (p *HTTPProvider) Present(domain, token, keyAuth string) error {
	if p.server != nil {
		return nil
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/acme-challenge/"+token, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(keyAuth))
	})

	p.server = &http.Server{
		Addr:    ":80",
		Handler: mux,
	}

	go func() {
		// 忽略 Server closed 錯誤
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("HTTP Provider Server Error: %v\n", err)
		}
	}()
	return nil
}

// CleanUp 關閉 HTTP 服務器
func (p *HTTPProvider) CleanUp(domain, token, keyAuth string) error {
	if p.server != nil {
		err := p.server.Close()
		p.server = nil
		return err
	}
	return nil
}
