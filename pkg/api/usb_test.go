package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/m-vinc/maco/pkg/auth"
	"github.com/m-vinc/maco/pkg/config"
	"github.com/m-vinc/maco/pkg/engine"
)

func TestUSBAuthenticationAndValidation(t *testing.T) {
	paths, err := config.Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	secret := []byte("usb-api-test-secret")
	server := New(paths, secret, nil, nil)
	token, err := issueAPITestToken(t, server)
	if err != nil {
		t.Fatal(err)
	}

	for _, route := range []string{"/api/usb/devices", "/api/vms/test/usb"} {
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, route, nil))
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("unprotected route %s", route)
		}
	}

	for _, body := range []string{`{}`, `{"device_id":"bad","fingerprint":"bad"}`, `{"hostbus":1,"hostaddr":2}`, `{"device_id":"` + strings.Repeat("a", 24) + `","fingerprint":"` + strings.Repeat("b", 64) + `","attachment_id":"injected"}`} {
		request := httptest.NewRequest(http.MethodPost, "/api/vms/test/usb", bytes.NewBufferString(body))
		request.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("accepted malformed selection %s: %d", body, recorder.Code)
		}
	}

	request := httptest.NewRequest(http.MethodDelete, "/api/vms/test/usb/usb-kbd", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatal("accepted unmanaged QEMU ID")
	}
}

func TestStoppedUSBSelectionCanBeQueued(t *testing.T) {
	paths, err := config.Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	secret := []byte("usb-offline-test-secret")
	server := New(paths, secret, nil, nil)
	m, err := server.engine.CreateVM(engine.CreateVMParams{Name: "usb-offline", Image: "ubuntu-24.04-arm64", CPUs: 1, MemoryMiB: 128, DiskSizeGiB: 4}, nil)
	if err != nil {
		t.Fatal(err)
	}

	token, err := issueAPITestToken(t, server)
	if err != nil {
		t.Fatal(err)
	}

	body := `{"device_id":"` + strings.Repeat("a", 24) + `","fingerprint":"` + strings.Repeat("b", 64) + `"}`
	request := httptest.NewRequest(http.MethodPost, "/api/vms/"+m.ID+"/usb/assignments", bytes.NewBufferString(body))
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable || !strings.Contains(recorder.Body.String(), "job queue unavailable") {
		t.Fatalf("stopped assignment rejected before queue: %d %s", recorder.Code, recorder.Body.String())
	}
}

func issueAPITestToken(t *testing.T, server *Server) (string, error) {
	t.Helper()
	if err := server.engine.AddUser(context.Background(), "test", "test-password", "admin"); err != nil {
		return "", err
	}

	user, err := server.engine.Authenticate(context.Background(), "test", "test-password")
	if err != nil {
		return "", err
	}

	return auth.IssueUserToken(server.secret, user, time.Hour)
}
