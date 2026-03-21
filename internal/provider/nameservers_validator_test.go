package provider

import (
	"testing"
)

func TestIsDefaultBasicNameservers_Match(t *testing.T) {
	hosts := []string{"launch1.spaceship.net", "launch2.spaceship.net"}
	if !isDefaultBasicNameservers(hosts) {
		t.Error("expected default hosts to match")
	}
}

func TestIsDefaultBasicNameservers_CaseInsensitive(t *testing.T) {
	hosts := []string{"LAUNCH1.SPACESHIP.NET", "Launch2.Spaceship.Net"}
	if !isDefaultBasicNameservers(hosts) {
		t.Error("expected case-insensitive match")
	}
}

func TestIsDefaultBasicNameservers_DifferentOrder(t *testing.T) {
	hosts := []string{"launch2.spaceship.net", "launch1.spaceship.net"}
	if !isDefaultBasicNameservers(hosts) {
		t.Error("expected match regardless of order")
	}
}

func TestIsDefaultBasicNameservers_NonDefault(t *testing.T) {
	hosts := []string{"ns1.example.com", "ns2.example.com"}
	if isDefaultBasicNameservers(hosts) {
		t.Error("expected non-default hosts to not match")
	}
}

func TestIsDefaultBasicNameservers_WrongCount(t *testing.T) {
	hosts := []string{"launch1.spaceship.net"}
	if isDefaultBasicNameservers(hosts) {
		t.Error("expected wrong count to not match")
	}
}

func TestIsDefaultBasicNameservers_Empty(t *testing.T) {
	if isDefaultBasicNameservers(nil) {
		t.Error("expected nil to not match")
	}
}
