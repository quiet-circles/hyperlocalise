package storageregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/storage"
)

type stubAdapter struct{}

func (stubAdapter) Name() string                       { return "stub" }
func (stubAdapter) Capabilities() storage.Capabilities { return storage.Capabilities{} }
func (stubAdapter) Pull(_ context.Context, _ storage.PullRequest) (storage.PullResult, error) {
	return storage.PullResult{}, nil
}

func (stubAdapter) Push(_ context.Context, _ storage.PushRequest) (storage.PushResult, error) {
	return storage.PushResult{}, nil
}

func TestRegistryRegisterAndNew(t *testing.T) {
	reg := New()
	if err := reg.Register("stub", func(_ json.RawMessage) (storage.StorageAdapter, error) {
		return stubAdapter{}, nil
	}); err != nil {
		t.Fatalf("register adapter: %v", err)
	}

	adapter, err := reg.New("stub", nil)
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	if got := adapter.Name(); got != "stub" {
		t.Fatalf("unexpected adapter name: %q", got)
	}
}

func TestRegistryDuplicateRegister(t *testing.T) {
	reg := New()
	factory := func(_ json.RawMessage) (storage.StorageAdapter, error) { return stubAdapter{}, nil }
	if err := reg.Register("stub", factory); err != nil {
		t.Fatalf("register first adapter: %v", err)
	}
	if err := reg.Register("stub", factory); err == nil || !strings.Contains(err.Error(), "already registered") {
		t.Fatalf("expected duplicate register error, got %v", err)
	}
}

func TestRegistryUnknownAdapter(t *testing.T) {
	reg := New()
	if _, err := reg.New("missing", nil); err == nil || !strings.Contains(err.Error(), "unknown adapter") {
		t.Fatalf("expected unknown adapter error, got %v", err)
	}
}

func TestRegistryListSorted(t *testing.T) {
	reg := New()
	factory := func(_ json.RawMessage) (storage.StorageAdapter, error) { return stubAdapter{}, nil }

	if err := reg.Register("zeta", factory); err != nil {
		t.Fatalf("register zeta: %v", err)
	}
	if err := reg.Register("alpha", factory); err != nil {
		t.Fatalf("register alpha: %v", err)
	}

	got := reg.List()
	want := []string{"alpha", "zeta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected registry list: got %v want %v", got, want)
	}
}

func TestRegistryMustRegisterPanicsOnDuplicate(t *testing.T) {
	reg := New()
	factory := func(_ json.RawMessage) (storage.StorageAdapter, error) { return stubAdapter{}, nil }
	reg.MustRegister("stub", factory)

	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic from duplicate MustRegister")
		}
		if msg := fmt.Sprint(r); !strings.Contains(msg, "already registered") {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()

	reg.MustRegister("stub", factory)
}
