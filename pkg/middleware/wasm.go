package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
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

func VerifyWasmHash(path, expectedHex string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	actual := hex.EncodeToString(h.Sum(nil))
	if actual != expectedHex {
		return fmt.Errorf("wasm hash mismatch: expected %s got %s", expectedHex, actual)
	}

	return nil
}
