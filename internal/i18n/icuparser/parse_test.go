package icuparser

import "testing"

func TestParseASTBasicElements(t *testing.T) {
	elems, err := Parse("Hi {name}", nil)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(elems) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(elems))
	}
	if elems[0].Type() != TypeLiteral {
		t.Fatalf("expected literal, got %s", elems[0].Type())
	}
	if elems[1].Type() != TypeArgument {
		t.Fatalf("expected argument, got %s", elems[1].Type())
	}
}

func TestParseASTPluralHasPound(t *testing.T) {
	elems, err := Parse("{count, plural, one {# item} other {# items}}", nil)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(elems) != 1 {
		t.Fatalf("expected 1 element, got %d", len(elems))
	}
	pl, ok := elems[0].(PluralElement)
	if !ok {
		t.Fatalf("expected plural element, got %T", elems[0])
	}
	if pl.Type() != TypePlural {
		t.Fatalf("unexpected plural type: %s", pl.Type())
	}
	if len(pl.Options) != 2 {
		t.Fatalf("expected 2 plural options, got %d", len(pl.Options))
	}
	foundPound := false
	for _, el := range pl.Options[0].Value {
		if el.Type() == TypePound {
			foundPound = true
			break
		}
	}
	if !foundPound {
		t.Fatalf("expected pound element in plural option")
	}
}

func TestParseASTTags(t *testing.T) {
	elems, err := Parse("Click <b>{name}</b> now", nil)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(elems) != 3 {
		t.Fatalf("expected 3 top-level elements, got %d", len(elems))
	}
	tag, ok := elems[1].(TagElement)
	if !ok {
		t.Fatalf("expected tag element, got %T", elems[1])
	}
	if tag.Value != "b" || tag.SelfClosing {
		t.Fatalf("unexpected tag: %+v", tag)
	}
	if len(tag.Children) != 1 || tag.Children[0].Type() != TypeArgument {
		t.Fatalf("unexpected tag children: %+v", tag.Children)
	}
}

func TestParseASTIgnoreTagOption(t *testing.T) {
	elems, err := Parse("<b>x</b>", &ParseOptions{IgnoreTag: true})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(elems) != 1 || elems[0].Type() != TypeLiteral {
		t.Fatalf("expected single literal, got %+v", elems)
	}
}
