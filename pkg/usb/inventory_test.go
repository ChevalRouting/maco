package usb

import "testing"

func TestConnectionIdentity(t *testing.T) {
	first := Device{Bus: 1, Address: 2, Port: "3.1", Session: "42", VendorID: 123, ProductID: 456, Classes: []string{}}
	identify(&first)
	second := first
	second.Port = "3.2"
	identify(&second)
	if first.ID == second.ID {
		t.Fatal("identical products at different ports must have different IDs")
	}

	replacement := first
	replacement.Session = "43"
	identify(&replacement)
	if first.ID == replacement.ID {
		t.Fatal("replacement at the same address must invalidate old selection")
	}

	if _, err := Resolve([]Device{replacement}, first.ID, first.Fingerprint); err == nil {
		t.Fatal("stale selection accepted")
	}

	if _, err := Resolve([]Device{first}, first.ID, second.Fingerprint); err == nil {
		t.Fatal("wrong fingerprint accepted")
	}

	if _, err := Resolve([]Device{first}, first.ID, first.Fingerprint); err != nil {
		t.Fatal(err)
	}
}

func TestInventoryEligibility(t *testing.T) {
	hub := Device{Bus: 1, Address: 2, Port: "1", Session: "42", Classes: []string{"Hub"}}
	identify(&hub)
	if _, err := Resolve([]Device{hub}, hub.ID, hub.Fingerprint); err == nil {
		t.Fatal("hub was eligible")
	}

	unknown := hub
	unknown.Classes = []string{}
	unknown.Port = ""
	identify(&unknown)
	if unknown.State != "unsupported" {
		t.Fatal("device without a location was eligible")
	}

	if got := cleanDescriptor("A\tB\n\x00"); got != "AB" {
		t.Fatalf("control characters remained: %q", got)
	}
}

func TestCaptureCanHideSerialDescriptor(t *testing.T) {
	selected := Device{Bus: 1, Address: 2, Port: "1", Session: "42", Serial: "unique", VendorID: 1, ProductID: 2}
	identify(&selected)
	captured := selected
	captured.Serial = ""
	identify(&captured)
	if err := VerifyConnection([]Device{captured}, selected); err != nil {
		t.Fatalf("same physical session rejected when QEMU holds device: %v", err)
	}

	captured.Session = "43"
	identify(&captured)
	if err := VerifyConnection([]Device{captured}, selected); err == nil {
		t.Fatal("replacement accepted after capture")
	}
}

func TestBusZeroRequiresUniqueQEMUSelector(t *testing.T) {
	selected := Device{Bus: 0, Address: 1, Port: "1", Session: "42", VendorID: 4176, ProductID: 1031}
	identify(&selected)
	if _, err := Resolve([]Device{selected}, selected.ID, selected.Fingerprint); err != nil {
		t.Fatalf("valid bus zero rejected: %v", err)
	}

	other := selected
	other.Bus = 1
	other.Session = "43"
	identify(&other)
	devices := []Device{selected, other}
	if _, err := Resolve(devices, selected.ID, selected.Fingerprint); err == nil {
		t.Fatal("ambiguous wildcard bus accepted")
	}

	if err := VerifyConnection(devices, selected); err == nil {
		t.Fatal("ambiguous post-capture selector accepted")
	}

	if _, err := Resolve(devices, other.ID, other.Fingerprint); err != nil {
		t.Fatalf("exact nonzero bus rejected: %v", err)
	}
}
