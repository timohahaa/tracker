package passwords

import "testing"

func TestHashVerify(t *testing.T) {
	const pw = "correct-horse-battery-staple"

	hash, err := Hash(pw)
	if err != nil {
		t.Fatalf("Hash: unexpected error: %v", err)
	}
	if hash == "" {
		t.Fatal("Hash returned empty string")
	}
	if hash == pw {
		t.Fatal("Hash returned the plaintext password")
	}

	if !Verify(hash, pw) {
		t.Error("Verify returned false for correct password")
	}
	if Verify(hash, "wrong-password") {
		t.Error("Verify returned true for wrong password")
	}
}

func TestHashDistinctForSamePassword(t *testing.T) {
	const pw = "same-password"

	h1, err := Hash(pw)
	if err != nil {
		t.Fatalf("Hash: unexpected error: %v", err)
	}
	h2, err := Hash(pw)
	if err != nil {
		t.Fatalf("Hash: unexpected error: %v", err)
	}
	if h1 == h2 {
		t.Error("two hashes of the same password are identical (missing salt?)")
	}
	// Оба хеша должны верифицироваться, несмотря на разную соль.
	if !Verify(h1, pw) || !Verify(h2, pw) {
		t.Error("Verify failed for a valid hash")
	}
}
