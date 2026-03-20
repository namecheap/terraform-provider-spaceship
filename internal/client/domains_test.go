package client

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestGetDomainList(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("take") != "100" {
			t.Errorf("expected take=100, got %q", r.URL.Query().Get("take"))
		}
		_ = json.NewEncoder(w).Encode(DomainList{
			Items: []DomainInfo{{Name: "example.com", AutoRenew: true}},
			Total: 1,
		})
	})

	list, err := c.GetDomainList(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("expected 1 domain, got %d", len(list.Items))
	}
	if list.Items[0].Name != "example.com" {
		t.Errorf("expected example.com, got %q", list.Items[0].Name)
	}
}

func TestGetDomainInfo_Success(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(DomainInfo{
			Name:      "example.com",
			AutoRenew: true,
		})
	})

	info, err := c.GetDomainInfo(t.Context(), "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Name != "example.com" {
		t.Errorf("expected example.com, got %q", info.Name)
	}
}

func TestGetDomainInfo_RateLimitFallback(t *testing.T) {
	callCount := 0
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte("rate limited"))
			return
		}
		// second call is GetDomainList fallback
		_ = json.NewEncoder(w).Encode(DomainList{
			Items: []DomainInfo{
				{Name: "other.com"},
				{Name: "example.com", AutoRenew: true},
			},
			Total: 2,
		})
	})

	info, err := c.GetDomainInfo(t.Context(), "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Name != "example.com" {
		t.Errorf("expected example.com, got %q", info.Name)
	}
	if !info.AutoRenew {
		t.Error("expected AutoRenew to be true from fallback")
	}
}

func TestGetDomainInfo_RateLimitFallback_NotFound(t *testing.T) {
	callCount := 0
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte("rate limited"))
			return
		}
		_ = json.NewEncoder(w).Encode(DomainList{
			Items: []DomainInfo{{Name: "other.com"}},
			Total: 1,
		})
	})

	// When rate limited and domain not found in list, the original 429 error is returned
	_, err := c.GetDomainInfo(t.Context(), "example.com")
	if err == nil {
		t.Fatal("expected error when domain not found in fallback list")
	}
}

func TestUpdateAutoRenew(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		var body struct {
			IsEnabled bool `json:"isEnabled"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		_ = json.NewEncoder(w).Encode(AutoRenewalResponse{IsEnabled: body.IsEnabled})
	})

	resp, err := c.UpdateAutoRenew(t.Context(), "example.com", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.IsEnabled {
		t.Error("expected IsEnabled to be true")
	}
}

func TestUpdateDomainNameServers(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		var body struct {
			Provider string   `json:"provider"`
			Hosts    []string `json:"hosts,omitempty"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Provider != "custom" {
			t.Errorf("expected provider=custom, got %q", body.Provider)
		}
		if len(body.Hosts) != 2 {
			t.Errorf("expected 2 hosts, got %d", len(body.Hosts))
		}
		w.WriteHeader(http.StatusOK)
	})

	err := c.UpdateDomainNameServers(t.Context(), "example.com", UpdateNameserverRequest{
		Provider: CustomNamerverProvider,
		Hosts:    []string{"ns1.example.com", "ns2.example.com"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNameserverProvider_Valid(t *testing.T) {
	if !BasicNameserverProvider.Valid() {
		t.Error("basic should be valid")
	}
	if !CustomNamerverProvider.Valid() {
		t.Error("custom should be valid")
	}
	if NameserverProvider("invalid").Valid() {
		t.Error("invalid should not be valid")
	}
}

func TestDefaultBasicNameserverHosts(t *testing.T) {
	hosts := DefaultBasicNameserverHosts()
	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(hosts))
	}
	if hosts[0] != "launch1.spaceship.net" || hosts[1] != "launch2.spaceship.net" {
		t.Errorf("unexpected default hosts: %v", hosts)
	}
}

func TestFindDomainByNameFromDomainList(t *testing.T) {
	dl := DomainList{
		Items: []DomainInfo{
			{Name: "first.com"},
			{Name: "target.com", AutoRenew: true},
			{Name: "third.com"},
		},
	}

	info, ok := findDomainByNameFromDomainList(dl, "target.com")
	if !ok {
		t.Fatal("expected to find target.com")
	}
	if !info.AutoRenew {
		t.Error("expected AutoRenew to be true")
	}

	_, ok = findDomainByNameFromDomainList(dl, "missing.com")
	if ok {
		t.Error("expected not to find missing.com")
	}
}
