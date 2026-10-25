package operations

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	s, err := New(filepath.Join(t.TempDir(), "operations.db"), time.Minute, 2)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s
}

func TestCacheEvictsAndLazilyRestoresWithinLimit(t *testing.T) {
	s := testStore(t)
	for _, id := range []string{"a", "b", "c"} {
		if err := s.Save(id, []byte(id)); err != nil {
			t.Fatal(err)
		}
	}
	if len(s.cache) != 2 {
		t.Fatal("unbounded cache")
	}
	s.Evict(time.Now().Add(2 * time.Minute))
	if len(s.cache) != 0 {
		t.Fatal("inactive cache remains")
	}
	for _, id := range []string{"a", "b", "c"} {
		data, err := s.Load(id)
		if err != nil || string(data) != id {
			t.Fatalf("restore %q: %s %v", id, data, err)
		}
		data[0] = 'x'
	}
	data, _ := s.Load("c")
	if string(data) != "c" || len(s.cache) > 2 {
		t.Fatal("aliased snapshot or restore bypasses cache limit")
	}
	if err := s.SaveSettings("channels", []byte(`{"ativo":false}`)); err != nil {
		t.Fatal(err)
	}
	settings, err := s.Settings("channels")
	if err != nil || !bytes.Contains(settings, []byte("false")) {
		t.Fatal(err)
	}
	s.Close()
	s.Close()
}

func TestIndividualDeliveryWritesAndSmallPaginatedHistory(t *testing.T) {
	s := testStore(t)
	if err := s.Save("agenda", []byte(`{"candidate":"preserved"}`)); err != nil {
		t.Fatal(err)
	}
	db, err := s.open()
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TRIGGER forbid_snapshot_rewrite BEFORE UPDATE ON agendas BEGIN SELECT RAISE(FAIL,'snapshot rewrite'); END;`)
	_ = db.Close()
	if err != nil {
		t.Fatal(err)
	}
	items := make([]Delivery, 1000)
	for i := range items {
		items[i] = Delivery{PersonID: i + 1, Name: "Synthetic candidate", Channel: "email", Message: string(bytes.Repeat([]byte("x"), 500)), Destination: "candidate@example.test"}
	}
	for i := 0; i < 12; i++ {
		if _, err := s.CreateBatch("agenda", items); err != nil {
			t.Fatal(err)
		}
	}
	page, err := s.History("agenda", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 10 || page.Next == 0 || page.Items[0].Pending != 1000 {
		t.Fatalf("invalid headers: %#v", page)
	}
	encoded, _ := json.Marshal(page)
	if len(encoded) > 3000 || bytes.Contains(encoded, []byte("Synthetic")) {
		t.Fatalf("history leaks recipient bodies: %d bytes", len(encoded))
	}
	older, err := s.History("agenda", page.Next, 10)
	if err != nil || len(older.Items) != 2 {
		t.Fatal("keyset pagination", err)
	}
	details, err := s.Deliveries("agenda", page.Items[0].ID, 0, 50)
	if err != nil || len(details.Items) != 50 || details.Next == 0 {
		t.Fatal("detail pagination", err)
	}
	claimed, err := s.Claim(details.Items[0].ID)
	if err != nil || !claimed {
		t.Fatal(err)
	}
	if err := s.Complete(details.Items[0].ID, nil); err != nil {
		t.Fatal("delivery rewrote agenda", err)
	}
	data, _ := s.Load("agenda")
	if !bytes.Contains(data, []byte("preserved")) {
		t.Fatal("agenda changed")
	}
	next, _ := s.Deliveries("agenda", page.Items[0].ID, details.Next, 50)
	if next.Items[0].ID <= details.Items[49].ID {
		t.Fatal("duplicate detail page")
	}
	other, _ := s.Deliveries("other-session", page.Items[0].ID, 0, 100)
	if len(other.Items) > 0 {
		t.Fatal("cross-session access")
	}
}

func TestClaimsRetryAndInterruptedRecovery(t *testing.T) {
	s := testStore(t)
	id, err := s.CreateBatch("agenda", []Delivery{{PersonID: 1, Channel: "email"}, {PersonID: 2, Channel: "email"}, {PersonID: 3, Channel: "email"}})
	if err != nil {
		t.Fatal(err)
	}
	page, _ := s.Deliveries("agenda", id, 0, 100)
	var claims atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := s.Claim(page.Items[0].ID)
			if err != nil {
				t.Error(err)
			}
			if ok {
				claims.Add(1)
			}
		}()
	}
	wg.Wait()
	if claims.Load() != 1 {
		t.Fatal("message claimed multiple times")
	}
	if err := s.Complete(page.Items[0].ID, nil); err != nil {
		t.Fatal(err)
	}
	_, _ = s.Claim(page.Items[1].ID)
	if err := s.Complete(page.Items[1].ID, errors.New("temporary failure")); err != nil {
		t.Fatal(err)
	}
	_, _ = s.Claim(page.Items[2].ID)
	if _, err := s.Retry("agenda", id); err == nil {
		t.Fatal("retry while sending")
	}
	if err := s.RecoverInterrupted(); err != nil {
		t.Fatal(err)
	}
	newID, err := s.Retry("agenda", id)
	if err != nil {
		t.Fatal(err)
	}
	retries, _ := s.Deliveries("agenda", newID, 0, 100)
	if len(retries.Items) != 1 || retries.Items[0].PersonID != 2 {
		t.Fatal("accepted or ambiguous delivery repeated")
	}
	if _, err := s.Retry("agenda", id); err == nil {
		t.Fatal("repeated failures duplicated")
	}
	if _, err := s.Retry("other", newID); err == nil {
		t.Fatal("cross-session retry")
	}
	states, err := s.NoticeStates("agenda")
	if err != nil || states[1] != "accepted" || states[2] != "pending" || states[3] != "unknown" {
		t.Fatal(states, err)
	}
}

func TestInvalidCacheAndEmptyBatch(t *testing.T) {
	if _, err := New(filepath.Join(t.TempDir(), "a"), 0, 0); err == nil {
		t.Fatal("invalid cache allowed")
	}
	s := testStore(t)
	if _, err := s.CreateBatch("agenda", nil); err == nil {
		t.Fatal("empty batch allowed")
	}
	if _, err := s.Load("missing"); err == nil {
		t.Fatal("missing snapshot allowed")
	}
}

func TestPendingJobsAndNoticeScheduleState(t *testing.T) {
	s := testStore(t)
	id, err := s.CreateBatch("agenda", []Delivery{{PersonID: 1, Channel: "email", Variables: map[string]string{"agenda_horario": "sexta 14-16"}}})
	if err != nil {
		t.Fatal(err)
	}
	pending, err := s.PendingBatches("agenda")
	if err != nil || len(pending) != 1 || pending[0] != id {
		t.Fatal(pending, err)
	}
	page, _ := s.Deliveries("agenda", id, 0, 100)
	_, _ = s.Claim(page.Items[0].ID)
	if err := s.Complete(page.Items[0].ID, nil); err != nil {
		t.Fatal(err)
	}
	states, err := s.NoticeStates("agenda", map[int]string{1: "sábado 10-12"})
	if err != nil || states[1] != "" {
		t.Fatal("old schedule appears notified", states, err)
	}
	pending, _ = s.PendingBatches("agenda")
	if len(pending) != 0 {
		t.Fatal("accepted job resumed")
	}
}

func TestUncertainNetworkResponseIsNotRetried(t *testing.T) {
	s := testStore(t)
	id, err := s.CreateBatch("agenda", []Delivery{{PersonID: 1, Channel: "email"}})
	if err != nil {
		t.Fatal(err)
	}
	page, _ := s.Deliveries("agenda", id, 0, 100)
	_, _ = s.Claim(page.Items[0].ID)
	if err := s.Complete(page.Items[0].ID, ErrUncertain); err != nil {
		t.Fatal(err)
	}
	history, err := s.History("agenda", 0, 10)
	if err != nil || history.Items[0].Unknown != 1 || history.Items[0].Failed != 0 {
		t.Fatal(history, err)
	}
	if _, err := s.Retry("agenda", id); err == nil {
		t.Fatal("ambiguous acceptance repeated")
	}
}
