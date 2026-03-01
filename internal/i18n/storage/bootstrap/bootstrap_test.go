package bootstrap

import (
	"reflect"
	"strings"
	"testing"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/storageregistry"
)

func TestRegisterBuiltinsRegistersExpectedAdapters(t *testing.T) {
	reg := storageregistry.New()

	if err := RegisterBuiltins(reg); err != nil {
		t.Fatalf("register built-ins: %v", err)
	}

	got := reg.List()
	want := []string{"crowdin", "lokalise", "poeditor", "smartling"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected registered adapters: got %v want %v", got, want)
	}
}

func TestRegisterBuiltinsRejectsDuplicates(t *testing.T) {
	reg := storageregistry.New()

	if err := RegisterBuiltins(reg); err != nil {
		t.Fatalf("register built-ins first time: %v", err)
	}
	if err := RegisterBuiltins(reg); err == nil || !strings.Contains(err.Error(), "already registered") {
		t.Fatalf("expected duplicate registration error, got %v", err)
	}
}
