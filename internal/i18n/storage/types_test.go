package storage

import "testing"

func TestEntryIDUsesKeyContextLocale(t *testing.T) {
	entry := Entry{
		Key:     "checkout.submit",
		Context: "button",
		Locale:  "fr",
		Value:   "Passer la commande",
	}

	got := entry.ID()
	want := EntryID{
		Key:     "checkout.submit",
		Context: "button",
		Locale:  "fr",
	}

	if got != want {
		t.Fatalf("unexpected entry id: got %+v want %+v", got, want)
	}
}
