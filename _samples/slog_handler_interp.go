// An interpreted type satisfying slog.Handler (and slog.LogValuer) across the
// native boundary: slog.New dispatches Enabled/Handle/WithAttrs/WithGroup via
// synth method stubs (shapes S32-S36).
package main

import (
	"context"
	"fmt"
	"log/slog"
)

type handler struct{ prefix string }

func (h *handler) Enabled(_ context.Context, level slog.Level) bool { return level >= slog.LevelInfo }
func (h *handler) Handle(ctx context.Context, r slog.Record) error {
	n := 0
	r.Attrs(func(a slog.Attr) bool { n++; return true })
	fmt.Printf("%s%s level=%v attrs=%d\n", h.prefix, r.Message, r.Level, n)
	return nil
}
func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &handler{prefix: h.prefix + fmt.Sprintf("[%d attrs]", len(attrs))}
}
func (h *handler) WithGroup(name string) slog.Handler {
	return &handler{prefix: h.prefix + "[" + name + "]"}
}

type addr struct {
	host string
	port int
}

func (a addr) LogValue() slog.Value {
	return slog.GroupValue(slog.String("host", a.host), slog.Int("port", a.port))
}

func main() {
	l := slog.New(&handler{})
	l.Info("hello", "k", "v")
	l.Debug("dropped")
	l.With("a", 1, "b", 2).WithGroup("grp").Warn("nested", "x", "y")

	v := slog.AnyValue(addr{host: "example.com", port: 443})
	fmt.Println(v.Resolve())
}

// Output:
// hello level=INFO attrs=1
// [2 attrs][grp]nested level=WARN attrs=1
// [host=example.com port=443]
