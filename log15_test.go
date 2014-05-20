package log15

import (
	"bytes"
	"encoding/json"
	"testing"
)

type testHandler struct {
	r Record
}

func (h *testHandler) Log(r *Record) error {
	h.r = *r
	return nil
}

func testLogger() (Logger, *Record) {
	l := New()
	h := &testHandler{}
	l.SetHandler(&lazyHandler{h})
	return l, &h.r
}

func TestLazy(t *testing.T) {
	t.Parallel()

	x := 1
	lazy := func() int {
		return x
	}

	l, r := testLogger()
	l.Info("", "x", Lazy(lazy))
	if r.Ctx[1] != 1 {
		t.Fatalf("Lazy function not evaluated, got %v, expected %d", r.Ctx[1], 1)
	}

	x = 2
	l.Info("", "x", Lazy(lazy))
	if r.Ctx[1] != 2 {
		t.Fatalf("Lazy function not evaluated, got %v, expected %d", r.Ctx[1], 1)
	}
}

func TestInvalidLazy(t *testing.T) {
	t.Parallel()

	l, r := testLogger()
	validate := func() {
		if len(r.Ctx) < 4 {
			t.Fatalf("Invalid lazy, got %d args, expecting at least 4", len(r.Ctx))
		}

		if r.Ctx[2] != errorKey {
			t.Fatalf("Invalid lazy, got key %s expecting %s", r.Ctx[2], errorKey)
		}
	}

	l.Info("", "x", Lazy(1))
	validate()

	l.Info("", "x", Lazy(func(x int) int { return x }))
	validate()

	l.Info("", "x", Lazy(func() {}))
	validate()
}

func TestCtx(t *testing.T) {
	t.Parallel()

	l, r := testLogger()
	l.Info("", Ctx{"x": 1, "y": "foo", "tester": t})
	if len(r.Ctx) != 6 {
		t.Fatalf("Expecting Ctx tansformed into %d ctx args, got %d: %v", 6, len(r.Ctx), r.Ctx)
	}
}

func testFormatter(f Format) (Logger, *bytes.Buffer) {
	l := New()
	var buf bytes.Buffer
	l.SetHandler(StreamHandler(&buf, f))
	return l, &buf
}

func TestJson(t *testing.T) {
	t.Parallel()

	l, buf := testFormatter(JsonFormat())
	l.Error("some message", "x", 1, "y", 3.2)

	var v map[string]interface{}
	decoder := json.NewDecoder(buf)
	if err := decoder.Decode(&v); err != nil {
		t.Fatalf("Error decoding JSON: %v", v)
	}

	validate := func(key string, expected interface{}) {
		if v[key] != expected {
			t.Fatalf("Got %v expected %v for %v", v[key], expected, key)
		}
	}

	validate("msg", "some message")
	validate("x", float64(1)) // all numbers are floats in JSON land
	validate("y", 3.2)
}

func TestLogfmt(t *testing.T) {
	t.Parallel()

	l, buf := testFormatter(LogfmtFormat())
	l.Error("some message", "x", 1, "y", 3.2, "equals", "=", "quote", "\"")

	// skip timestamp in comparison
	got := buf.Bytes()[27:buf.Len()]
	expected := []byte(`lvl=eror msg="some message" x=1 y=3.200 equals="=" quote="\"" ` + "\n")
	if !bytes.Equal(got, expected) {
		t.Fatalf("Got %s, expected %s", got, expected)
	}
}

func TestMultiHandler(t *testing.T) {
	t.Parallel()

	h1, h2 := &testHandler{}, &testHandler{}
	l := New()
	l.SetHandler(MultiHandler(h1, h2))
	l.Debug("clone")

	if h1.r.Msg != "clone" {
		t.Fatalf("wrong value for h1.Msg. Got %s expected %s", h1.r.Msg, "clone")
	}

	if h2.r.Msg != "clone" {
		t.Fatalf("wrong value for h2.Msg. Got %s expected %s", h2.r.Msg, "clone")
	}

}

func TestLogContext(t *testing.T) {
	l, r := testLogger()
	l = l.New("foo", "bar")
	l.Crit("baz")

	if len(r.Ctx) != 2 {
		t.Fatalf("Expected logger context in record context. Got length %d, expected %d", len(r.Ctx), 2)
	}

	if r.Ctx[0] != "foo" {
		t.Fatalf("Wrong context key, got %s expected %s", r.Ctx[0], "foo")
	}

	if r.Ctx[1] != "bar" {
		t.Fatalf("Wrong context value, got %s expected %s", r.Ctx[1], "bar")
	}
}