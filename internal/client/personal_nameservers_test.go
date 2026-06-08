package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// List returns every host with its glue IPs.
func TestListPersonalNameservers(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/domains/example.com/personal-nameservers") {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(PersonalNameserverList{
			Records: []PersonalNameserver{
				{Host: "ns1", IPs: []string{"1.2.3.4"}},
				{Host: "ns2", IPs: []string{"5.6.7.8"}},
			},
		})
	})

	list, err := c.ListPersonalNameservers(t.Context(), "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list.Records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(list.Records))
	}
}

// Find locates a host via the list endpoint, case-insensitively (501 workaround).
func TestFindPersonalNameserver_Found(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(PersonalNameserverList{
			Records: []PersonalNameserver{
				{Host: "ns1", IPs: []string{"1.2.3.4", "2001:db8::1"}},
			},
		})
	})

	ns, err := c.FindPersonalNameserver(t.Context(), "example.com", "NS1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ns.Host != "ns1" || len(ns.IPs) != 2 {
		t.Errorf("unexpected result: %+v", ns)
	}
}

// Find returns ErrPersonalNameserverNotFound when no host matches.
func TestFindPersonalNameserver_NotFound(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(PersonalNameserverList{
			Records: []PersonalNameserver{{Host: "ns1", IPs: []string{"1.2.3.4"}}},
		})
	})

	_, err := c.FindPersonalNameserver(t.Context(), "example.com", "ns9")
	if !errors.Is(err, ErrPersonalNameserverNotFound) {
		t.Fatalf("expected ErrPersonalNameserverNotFound, got %v", err)
	}
}

// Upsert sends host+ips in the body and the path host in the URL.
func TestUpsertPersonalNameserver(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/personal-nameservers/ns1") {
			t.Errorf("expected current host in path, got %q", r.URL.Path)
		}
		var body PersonalNameserver
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Host != "ns2" {
			t.Errorf("expected renamed host in body, got %q", body.Host)
		}
		_ = json.NewEncoder(w).Encode(body)
	})

	result, err := c.UpsertPersonalNameserver(t.Context(), "example.com", "ns1",
		PersonalNameserver{Host: "ns2", IPs: []string{"1.2.3.4"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Host != "ns2" {
		t.Errorf("expected ns2, got %q", result.Host)
	}
}

// Delete issues a DELETE on the host path.
func TestDeletePersonalNameserver(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/personal-nameservers/ns1") {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	if err := c.DeletePersonalNameserver(t.Context(), "example.com", "ns1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Delete swallows 404 so an already-deleted host still removes from state.
func TestDeletePersonalNameserver_NotFoundIgnored(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	})

	if err := c.DeletePersonalNameserver(t.Context(), "example.com", "ns1"); err != nil {
		t.Fatalf("expected 404 to be ignored, got: %v", err)
	}
}

// API errors propagate as SpaceshipApiError.
func TestPersonalNameserver_APIError(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte("bad request"))
	})

	_, err := c.UpsertPersonalNameserver(t.Context(), "example.com", "ns1",
		PersonalNameserver{Host: "ns1", IPs: []string{"1.2.3.4"}})
	var apiErr *SpaceshipApiError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 SpaceshipApiError, got %v", err)
	}
}

// validIPs returns n distinct, well-formed IPv4 addresses.
func validIPs(n int) []string {
	ips := make([]string, n)
	for i := range ips {
		ips[i] = fmt.Sprintf("10.0.0.%d", i+1)
	}
	return ips
}

func TestPersonalNameserver_Validate(t *testing.T) {
	tests := []struct {
		name    string
		ns      PersonalNameserver
		wantErr bool
	}{
		{"valid", PersonalNameserver{Host: "ns1", IPs: []string{"1.2.3.4", "2001:db8::1"}}, false},
		{"empty host", PersonalNameserver{Host: "", IPs: []string{"1.2.3.4"}}, true},
		{"apex host rejected", PersonalNameserver{Host: "@", IPs: []string{"1.2.3.4"}}, true},
		{"wildcard host rejected", PersonalNameserver{Host: "*", IPs: []string{"1.2.3.4"}}, true},
		{"no ips", PersonalNameserver{Host: "ns1", IPs: nil}, true},
		{"too many ips", PersonalNameserver{Host: "ns1", IPs: validIPs(17)}, true},
		{"invalid ip", PersonalNameserver{Host: "ns1", IPs: []string{"not-an-ip"}}, true},
		{"host too long", PersonalNameserver{Host: strings.Repeat("a", 256), IPs: []string{"1.2.3.4"}}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := tc.ns.Validate()
			if tc.wantErr && len(errs) == 0 {
				t.Error("expected validation error, got none")
			}
			if !tc.wantErr && len(errs) != 0 {
				t.Errorf("expected no error, got %v", errs)
			}
		})
	}
}
