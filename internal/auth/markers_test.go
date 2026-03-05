package auth

import "testing"

func TestMarkers(t *testing.T) {
	if !IsLoggedInBody([]byte(`<a id="_tlogout" href="/?action=logout">Logout</a>`)) {
		t.Fatal("expected logged in marker")
	}
	if !IsLoggedInBody([]byte(`<table id="domains_table"><tr><td>zone</td></tr></table>`)) {
		t.Fatal("expected logged in marker via domains table")
	}
	if !IsLoginPage([]byte(`Free DNS Login`)) {
		t.Fatal("expected login marker")
	}
	if IsLoggedInBody([]byte(`Free DNS Login`)) {
		t.Fatal("login page must not be treated as logged in")
	}
}
