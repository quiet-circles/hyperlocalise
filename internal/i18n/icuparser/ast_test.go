package icuparser

import "testing"

func TestElementTypeMethods(t *testing.T) {
	tests := []struct {
		name string
		el   Element
		want ElementType
	}{
		{name: "number", el: NumberElement{}, want: TypeNumber},
		{name: "date", el: DateElement{}, want: TypeDate},
		{name: "time", el: TimeElement{}, want: TypeTime},
		{name: "select", el: SelectElement{}, want: TypeSelect},
		{name: "tag", el: TagElement{}, want: TypeTag},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.el.Type(); got != tt.want {
				t.Fatalf("unexpected type: got %q want %q", got, tt.want)
			}
		})
	}
}
