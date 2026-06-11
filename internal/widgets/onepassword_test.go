package widgets

import "testing"

func TestParseOnePasswordAccounts(t *testing.T) {
	data := []byte(`[
		{"url":"example.1password.com","email":"a@example.com","user_uuid":"U1","account_uuid":"A1"},
		{"url":"team.1password.com","email":"","user_uuid":"U2","account_uuid":"A2"}
	]`)

	accounts, err := parseOnePasswordAccounts(data)

	if err != nil {
		t.Fatal(err)
	}

	if len(accounts) != 2 {
		t.Fatalf("accounts = %+v", accounts)
	}

	if accounts[0].id != "A1" || accounts[0].label != "a@example.com" {
		t.Fatalf("account 0 = %+v", accounts[0])
	}

	if accounts[1].label != "team.1password.com" {
		t.Fatalf("account 1 should fall back to url, got %q", accounts[1].label)
	}
}

func TestParseOnePasswordItems(t *testing.T) {
	data := []byte(`[
		{"id":"i1","title":"Example Login","vault":{"name":"Private"},"additional_information":"user@example.com"},
		{"id":"i2","title":"Empty Sub","vault":{"name":"Work"},"additional_information":"—"}
	]`)

	withVault, err := parseOnePasswordItems(data, opAccount{id: "A1", label: "a@example.com"}, false)

	if err != nil {
		t.Fatal(err)
	}

	if len(withVault) != 2 {
		t.Fatalf("items = %+v", withVault)
	}

	if withVault[0].id != "i1" || withVault[0].account != "A1" {
		t.Fatalf("item 0 = %+v", withVault[0])
	}

	if withVault[0].username != "user@example.com" || withVault[0].subtitle != "user@example.com · Private" {
		t.Fatalf("item 0 subtitle = %q", withVault[0].subtitle)
	}

	if withVault[1].username != "" || withVault[1].subtitle != "Work" {
		t.Fatalf("em-dash username should be dropped, got %+v", withVault[1])
	}

	withAccount, _ := parseOnePasswordItems(data, opAccount{id: "A1", label: "a@example.com"}, true)

	if withAccount[0].subtitle != "user@example.com · a@example.com" {
		t.Fatalf("multi-account subtitle = %q", withAccount[0].subtitle)
	}
}
