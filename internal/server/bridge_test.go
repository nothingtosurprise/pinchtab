package server

import (
	"testing"

	"github.com/pinchtab/pinchtab/internal/config"
	"github.com/pinchtab/pinchtab/internal/handlers"
)

func TestConfigureBridgeRouter(t *testing.T) {
	tests := []struct {
		name       string
		engine     string
		wantRouter bool
	}{
		{name: "chrome", engine: "chrome", wantRouter: false},
		{name: "lite", engine: "lite", wantRouter: true},
		{name: "auto", engine: "auto", wantRouter: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := handlers.New(nil, &config.RuntimeConfig{Engine: tt.engine}, nil, nil, nil)
			configureBridgeRouter(h, &config.RuntimeConfig{Engine: tt.engine})
			if (h.Router != nil) != tt.wantRouter {
				t.Fatalf("router presence = %v, want %v", h.Router != nil, tt.wantRouter)
			}
			if h.Router != nil && string(h.Router.Mode()) != tt.engine {
				t.Fatalf("router mode = %q, want %q", h.Router.Mode(), tt.engine)
			}
		})
	}
}
