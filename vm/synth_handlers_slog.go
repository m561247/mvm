package vm

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"unsafe"

	"github.com/mvm-sh/mvm/stdlib/stubs"
)

// Handlers for the log/slog method shapes S32-S35 (slog.Handler). Each
// re-enters the interpreter via callMethod, then marshals the result(s) back
// to the native return types.

// ctxArg boxes a context.Context as an interface-typed reflect value, keeping
// the interface type even for a nil ctx (ValueOf would yield an invalid Value).
func ctxArg(ctx context.Context) reflect.Value {
	return reflect.ValueOf(&ctx).Elem()
}

// makeHandlerS32 bridges S32: (T).Enabled(context.Context, slog.Level) bool.
func makeHandlerS32(m *Machine, t *Type, method Method, name string, ptrRecv bool) stubs.HandlerS32 {
	methodSig := method.Rtype
	return func(recv unsafe.Pointer, ctx context.Context, level slog.Level) bool {
		rv := makeRecvValue(t.Rtype, recv, ptrRecv)
		argv := []reflect.Value{ctxArg(ctx), reflect.ValueOf(level)}
		out, err := callMethod(m, t, name, rv, method, methodSig, argv)
		if err != nil || len(out) != 1 {
			return false
		}
		return out[0].Bool()
	}
}

// makeHandlerS33 bridges S33: (T).Handle(context.Context, slog.Record) error.
func makeHandlerS33(m *Machine, t *Type, method Method, name string, ptrRecv bool) stubs.HandlerS33 {
	methodSig := method.Rtype
	return func(recv unsafe.Pointer, ctx context.Context, record slog.Record) error {
		rv := makeRecvValue(t.Rtype, recv, ptrRecv)
		argv := []reflect.Value{ctxArg(ctx), reflect.ValueOf(record)}
		out, err := callMethod(m, t, name, rv, method, methodSig, argv)
		if err != nil {
			return err
		}
		if len(out) != 1 {
			return errors.New("synth: S33 dispatch produced wrong arity")
		}
		return reflectToError(out[0])
	}
}

// makeHandlerS34 bridges S34: (T).WithAttrs([]slog.Attr) slog.Handler.
func makeHandlerS34(m *Machine, t *Type, method Method, name string, ptrRecv bool) stubs.HandlerS34 {
	methodSig := method.Rtype
	return func(recv unsafe.Pointer, attrs []slog.Attr) slog.Handler {
		rv := makeRecvValue(t.Rtype, recv, ptrRecv)
		out, err := callMethod(m, t, name, rv, method, methodSig, []reflect.Value{reflect.ValueOf(attrs)})
		if err != nil || len(out) != 1 {
			return nil
		}
		h, _ := ifaceResult(out[0]).(slog.Handler)
		return h
	}
}

// makeHandlerS36 bridges S36: (T).LogValue() slog.Value (slog.LogValuer).
func makeHandlerS36(m *Machine, t *Type, method Method, name string, ptrRecv bool) stubs.HandlerS36 {
	methodSig := method.Rtype
	return func(recv unsafe.Pointer) slog.Value {
		rv := makeRecvValue(t.Rtype, recv, ptrRecv)
		out, err := callMethod(m, t, name, rv, method, methodSig, nil)
		if err != nil || len(out) != 1 {
			return slog.Value{}
		}
		v, _ := ifaceResult(out[0]).(slog.Value)
		return v
	}
}

// makeHandlerS35 bridges S35: (T).WithGroup(string) slog.Handler.
func makeHandlerS35(m *Machine, t *Type, method Method, name string, ptrRecv bool) stubs.HandlerS35 {
	methodSig := method.Rtype
	return func(recv unsafe.Pointer, name2 string) slog.Handler {
		rv := makeRecvValue(t.Rtype, recv, ptrRecv)
		out, err := callMethod(m, t, name, rv, method, methodSig, []reflect.Value{reflect.ValueOf(name2)})
		if err != nil || len(out) != 1 {
			return nil
		}
		h, _ := ifaceResult(out[0]).(slog.Handler)
		return h
	}
}
