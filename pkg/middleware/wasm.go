package middleware

import (
	"context"
	"net/http"
	"os"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

type WasmMiddleware struct {
	runtime wazero.Runtime
	module  wazero.CompiledModule
	next    http.Handler
}

func NewWasmMiddleware(ctx context.Context, wasmPath string, next http.Handler) (*WasmMiddleware, error) {
	r := wazero.NewRuntime(ctx)

	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		return nil, err
	}

	compiled, err := r.CompileModule(ctx, wasmBytes)
	if err != nil {
		return nil, err
	}

	return &WasmMiddleware{
		runtime: r,
		module:  compiled,
		next:    next,
	}, nil
}

func (m *WasmMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	mod, err := m.runtime.InstantiateModule(ctx, m.module, wazero.NewModuleConfig().WithName("plugin").WithStdout(os.Stdout))
	if err == nil {
		fn := mod.ExportedFunction("process_request")
		if fn != nil {
			_, _ = fn.Call(ctx)
		}
		mod.Close(ctx)
	}

	if m.next != nil {
		m.next.ServeHTTP(w, r)
	}
}
