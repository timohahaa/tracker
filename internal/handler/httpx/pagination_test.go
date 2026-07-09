package httpx

import "testing"

func TestEncodeDecodeCursorRoundTrip(t *testing.T) {
	const (
		value = "2026-07-07T12:00:00Z"
		id    = "0192f0a0-0000-7000-8000-000000000000"
	)

	raw := EncodeCursor(value, id)
	if raw == "" {
		t.Fatal("EncodeCursor returned empty string")
	}

	cur, err := DecodeCursor(raw)
	if err != nil {
		t.Fatalf("DecodeCursor: unexpected error: %v", err)
	}
	if cur == nil {
		t.Fatal("DecodeCursor returned nil cursor")
	}
	if cur.Value != value {
		t.Errorf("cursor value = %q, want %q", cur.Value, value)
	}
	if cur.ID != id {
		t.Errorf("cursor id = %q, want %q", cur.ID, id)
	}
}

func TestEncodeDecodeCursorPriorityRank(t *testing.T) {
	// Для sort=priority значением курсора выступает ранг CASE (int-строка).
	const (
		value = "3"
		id    = "0192f0a0-0000-7000-8000-000000000001"
	)

	cur, err := DecodeCursor(EncodeCursor(value, id))
	if err != nil {
		t.Fatalf("DecodeCursor: unexpected error: %v", err)
	}
	if cur == nil {
		t.Fatal("DecodeCursor returned nil cursor")
	}
	if cur.Value != value || cur.ID != id {
		t.Errorf("cursor = (%q, %q), want (%q, %q)", cur.Value, cur.ID, value, id)
	}
}

func TestDecodeCursorEmpty(t *testing.T) {
	cur, err := DecodeCursor("")
	if err != nil {
		t.Fatalf("DecodeCursor(\"\"): unexpected error: %v", err)
	}
	if cur != nil {
		t.Errorf("DecodeCursor(\"\") = %+v, want nil", cur)
	}
}

func TestDecodeCursorInvalidBase64(t *testing.T) {
	if _, err := DecodeCursor("!!!not-base64!!!"); err == nil {
		t.Fatal("DecodeCursor: expected error for invalid base64, got nil")
	}
}
