package safety

import (
	"testing"
	"time"
)

// --- Task 12.4: 预演生成后 5min 内 confirm → 执行成功 ---

func TestPreviewConfirmWithinTTL(t *testing.T) {
	engine := NewPreviewEngine()
	p := engine.Create("truncate -s 0 /var/log/app.log", "清空应用日志", "medium")

	if p.ID == "" {
		t.Fatal("expected non-empty preview ID")
	}
	if p.Status != PreviewPending {
		t.Fatalf("expected status pending, got %s", p.Status)
	}

	// Confirm within TTL
	confirmed, err := engine.Confirm(p.ID, true)
	if err != nil {
		t.Fatalf("unexpected error on confirm: %v", err)
	}
	if confirmed.Status != PreviewConfirmed {
		t.Fatalf("expected status confirmed, got %s", confirmed.Status)
	}

	t.Logf("✅ Preview confirmed successfully within TTL: %s → %s", p.ID, confirmed.Status)
}

// --- Task 12.5: 预演过期后 confirm → PREVIEW_EXPIRED_001 ---

func TestPreviewExpiredConfirm(t *testing.T) {
	engine := NewPreviewEngine()
	p := engine.Create("rm -rf /tmp/old_cache", "清理缓存", "low")

	// Manually expire the preview by setting ExpiresAt to the past
	engine.mu.Lock()
	engine.previews[p.ID].ExpiresAt = time.Now().Add(-1 * time.Minute)
	engine.mu.Unlock()

	// Try to confirm — should get PREVIEW_EXPIRED_001
	_, err := engine.Confirm(p.ID, true)
	if err == nil {
		t.Fatal("expected error for expired preview, got nil")
	}
	if err.Error() != "PREVIEW_EXPIRED_001" {
		t.Fatalf("expected PREVIEW_EXPIRED_001, got %q", err.Error())
	}

	// Check status is expired
	got, ok := engine.Get(p.ID)
	if !ok {
		t.Fatal("preview should still exist after expiry")
	}
	if got.Status != PreviewExpired {
		t.Fatalf("expected status expired, got %s", got.Status)
	}

	t.Logf("✅ Expired preview correctly returns PREVIEW_EXPIRED_001")
}

// --- Task 12.6: confirm false → status=cancelled 且不执行 ---

func TestPreviewCancelled(t *testing.T) {
	engine := NewPreviewEngine()
	p := engine.Create("systemctl restart nginx", "重启nginx", "medium")

	// Cancel instead of confirm
	cancelled, err := engine.Confirm(p.ID, false)
	if err != nil {
		t.Fatalf("unexpected error on cancel: %v", err)
	}
	if cancelled.Status != PreviewCancelled {
		t.Fatalf("expected status cancelled, got %s", cancelled.Status)
	}

	// Try to confirm after cancel — should fail
	_, err = engine.Confirm(p.ID, true)
	if err == nil {
		t.Fatal("expected error when confirming already cancelled preview")
	}

	t.Logf("✅ Preview cancelled correctly: %s → %s", p.ID, cancelled.Status)
}

// --- Additional: confirm non-existent preview ---

func TestPreviewNotFound(t *testing.T) {
	engine := NewPreviewEngine()
	_, err := engine.Confirm("prv_nonexistent", true)
	if err == nil {
		t.Fatal("expected error for non-existent preview")
	}
	t.Logf("✅ Non-existent preview returns error: %v", err)
}

// --- Additional: double confirm ---

func TestPreviewDoubleConfirm(t *testing.T) {
	engine := NewPreviewEngine()
	p := engine.Create("df -h", "查看磁盘", "low")

	// First confirm
	_, err := engine.Confirm(p.ID, true)
	if err != nil {
		t.Fatalf("first confirm failed: %v", err)
	}

	// Second confirm should fail
	_, err = engine.Confirm(p.ID, true)
	if err == nil {
		t.Fatal("expected error on double confirm")
	}

	t.Logf("✅ Double confirm correctly rejected: %v", err)
}
