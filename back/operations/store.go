// Package operations persists agendas and individual notification deliveries.
package operations

import (
	"candidate_alocator/back/diagnostics"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type cached struct {
	data     []byte
	accessed time.Time
	version  int64
}

// Store keeps only bounded, disposable agenda snapshots in memory. Database
// connections are scoped to each operation; delivery history is never cached.
type Store struct {
	path  string
	mu    sync.Mutex
	cache map[string]cached
	ttl   time.Duration
	limit int
	done  chan struct{}
	once  sync.Once
}

type Delivery struct {
	ID          int64             `json:"id"`
	BatchID     int64             `json:"batchId"`
	PersonID    int               `json:"personId"`
	GroupID     int               `json:"groupId"`
	Channel     string            `json:"channel"`
	Destination string            `json:"destination"`
	Name        string            `json:"name"`
	Message     string            `json:"message"`
	Variables   map[string]string `json:"variables"`
	Status      string            `json:"status"`
	Error       string            `json:"error"`
}

type Batch struct {
	ID         int64  `json:"id"`
	Created    string `json:"created"`
	Total      int    `json:"total"`
	Accepted   int    `json:"accepted"`
	Failed     int    `json:"failed"`
	Pending    int    `json:"pending"`
	Unknown    int    `json:"unknown"`
	RepeatedBy int64  `json:"repeatedBy"`
}

type HistoryPage struct {
	Items []Batch `json:"items"`
	Next  int64   `json:"next"`
}
type DeliveryPage struct {
	Items []Delivery `json:"items"`
	Next  int64      `json:"next"`
}

// ErrUncertain means the provider may have accepted a message even though its
// response was lost. Such a delivery must not enter the automatic failure retry.
var ErrUncertain = errors.New("aceitação não confirmada; confira no provedor antes de reenviar")

func New(path string, ttl time.Duration, limit int) (*Store, error) {
	if ttl <= 0 || limit < 1 {
		return nil, errors.New("limites de cache inválidos")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	// #nosec G304 -- application-owned local database path, not a remote request.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	s := &Store{path: path, cache: map[string]cached{}, ttl: ttl, limit: limit, done: make(chan struct{})}
	db, err := s.open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = db.Close() }()
	_, err = db.Exec(`
CREATE TABLE IF NOT EXISTS agendas(id TEXT PRIMARY KEY, data BLOB NOT NULL);
CREATE TABLE IF NOT EXISTS settings(id TEXT PRIMARY KEY, data BLOB NOT NULL);
CREATE TABLE IF NOT EXISTS batches(id INTEGER PRIMARY KEY AUTOINCREMENT, session TEXT NOT NULL, created TEXT NOT NULL, repeated_by INTEGER NOT NULL DEFAULT 0);
CREATE INDEX IF NOT EXISTS batches_session ON batches(session,id);
CREATE TABLE IF NOT EXISTS deliveries(id INTEGER PRIMARY KEY AUTOINCREMENT, batch_id INTEGER NOT NULL REFERENCES batches(id), payload BLOB NOT NULL, status TEXT NOT NULL, error TEXT NOT NULL DEFAULT '', updated TEXT NOT NULL);
CREATE INDEX IF NOT EXISTS deliveries_batch_status ON deliveries(batch_id,status,id);`)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec("ALTER TABLE agendas ADD COLUMN version INTEGER NOT NULL DEFAULT 0"); err != nil && !strings.Contains(err.Error(), "duplicate column") {
		return nil, err
	}
	go s.cleanup()
	return s, nil
}

func (s *Store) open() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "file:"+filepath.ToSlash(s.path)+"?_busy_timeout=5000&_foreign_keys=on")
	if err == nil {
		db.SetMaxOpenConns(1)
	}
	return db, err
}

func (s *Store) cleanup() {
	ticker := time.NewTicker(min(s.ttl, time.Minute))
	defer ticker.Stop()
	for {
		select {
		case now := <-ticker.C:
			s.Evict(now)
		case <-s.done:
			return
		}
	}
}

func (s *Store) Close() {
	s.once.Do(func() { close(s.done); s.mu.Lock(); s.cache = map[string]cached{}; s.mu.Unlock() })
}

func (s *Store) Evict(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, c := range s.cache {
		if now.Sub(c.accessed) >= s.ttl {
			delete(s.cache, id)
		}
	}
}

func (s *Store) cacheLocked(id string, data []byte, version int64) {
	if _, exists := s.cache[id]; !exists && len(s.cache) >= s.limit {
		var oldest string
		var at time.Time
		for k, c := range s.cache {
			if oldest == "" || c.accessed.Before(at) {
				oldest, at = k, c.accessed
			}
		}
		delete(s.cache, oldest)
	}
	s.cache[id] = cached{data: append([]byte(nil), data...), accessed: time.Now(), version: version}
}

// NoticeStates reads only the newest delivery per person and channel, rather
// than sending the history bodies to the result screen.
func (s *Store) NoticeStates(session string, schedules ...map[int]string) (map[int]string, error) {
	db, err := s.open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = db.Close() }()
	rows, err := db.Query(`SELECT json_extract(d.payload,'$.personId'),d.status,COALESCE(json_extract(d.payload,'$.variables.agenda_horario'),'') FROM deliveries d JOIN batches b ON b.id=d.batch_id
WHERE b.session=? AND d.id IN (SELECT MAX(x.id) FROM deliveries x JOIN batches y ON y.id=x.batch_id WHERE y.session=? GROUP BY json_extract(x.payload,'$.personId'),json_extract(x.payload,'$.channel'))`, session, session)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[int]string{}
	priority := map[string]int{"accepted": 1, "unknown": 2, "pending": 3, "sending": 3, "failed": 4}
	for rows.Next() {
		var id int
		var status, schedule string
		if err := rows.Scan(&id, &status, &schedule); err != nil {
			return nil, err
		}
		if len(schedules) > 0 && schedules[0][id] != schedule {
			continue
		}
		if priority[status] > priority[out[id]] {
			out[id] = status
		}
	}
	return out, rows.Err()
}

func (s *Store) Save(id string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	db, err := s.open()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	var version int64
	err = db.QueryRow("INSERT INTO agendas(id,data,version) VALUES(?,?,1) ON CONFLICT(id) DO UPDATE SET data=excluded.data,version=agendas.version+1 RETURNING version", id, data).Scan(&version)
	if err == nil {
		s.cacheLocked(id, data, version)
	}
	return err
}

func (s *Store) Load(id string) ([]byte, error) {
	defer diagnostics.Begin("agenda.load", 1)()
	s.mu.Lock()
	defer s.mu.Unlock()
	db, err := s.open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = db.Close() }()
	var version int64
	if err := db.QueryRow("SELECT version FROM agendas WHERE id=?", id).Scan(&version); err != nil {
		return nil, err
	}
	if c, ok := s.cache[id]; ok && c.version == version {
		c.accessed = time.Now()
		s.cache[id] = c
		return append([]byte(nil), c.data...), nil
	}
	var data []byte
	err = db.QueryRow("SELECT data,version FROM agendas WHERE id=?", id).Scan(&data, &version)
	if err == nil {
		s.cacheLocked(id, data, version)
	}
	return data, err
}

func (s *Store) SaveSettings(id string, data []byte) error {
	db, err := s.open()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	_, err = db.Exec("INSERT INTO settings(id,data) VALUES(?,?) ON CONFLICT(id) DO UPDATE SET data=excluded.data", id, data)
	return err
}

func (s *Store) Settings(id string) ([]byte, error) {
	db, err := s.open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = db.Close() }()
	var data []byte
	err = db.QueryRow("SELECT data FROM settings WHERE id=?", id).Scan(&data)
	return data, err
}

func (s *Store) CreateBatch(session string, items []Delivery) (int64, error) {
	if len(items) == 0 {
		return 0, errors.New("nenhum destinatário selecionado")
	}
	db, err := s.open()
	if err != nil {
		return 0, err
	}
	defer func() { _ = db.Close() }()
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	id, err := insertBatch(tx, session, items)
	if err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

func insertBatch(tx *sql.Tx, session string, items []Delivery) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	r, err := tx.Exec("INSERT INTO batches(session,created) VALUES(?,?)", session, now)
	if err != nil {
		return 0, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return 0, err
	}
	for _, item := range items {
		status := "pending"
		if item.Error != "" {
			status = "failed"
		}
		item.ID = 0
		item.BatchID = id
		item.Status = status
		data, err := json.Marshal(item)
		if err != nil {
			return 0, err
		}
		if _, err = tx.Exec("INSERT INTO deliveries(batch_id,payload,status,error,updated) VALUES(?,?,?,?,?)", id, data, status, item.Error, now); err != nil {
			return 0, err
		}
	}
	return id, nil
}

func (s *Store) History(session string, before int64, limit int) (HistoryPage, error) {
	limit = max(1, min(limit, 50))
	db, err := s.open()
	if err != nil {
		return HistoryPage{}, err
	}
	defer func() { _ = db.Close() }()
	rows, err := db.Query(`SELECT b.id,b.created,b.repeated_by,COUNT(d.id),COALESCE(SUM(d.status='accepted'),0),COALESCE(SUM(d.status='failed'),0),COALESCE(SUM(d.status IN ('pending','sending')),0),COALESCE(SUM(d.status='unknown'),0)
FROM (SELECT id,created,repeated_by FROM batches WHERE session=? AND (?=0 OR id<?) ORDER BY id DESC LIMIT ?) b
LEFT JOIN deliveries d ON d.batch_id=b.id GROUP BY b.id ORDER BY b.id DESC`, session, before, before, limit+1)
	if err != nil {
		return HistoryPage{}, err
	}
	defer func() { _ = rows.Close() }()
	page := HistoryPage{Items: []Batch{}}
	for rows.Next() {
		var b Batch
		if err := rows.Scan(&b.ID, &b.Created, &b.RepeatedBy, &b.Total, &b.Accepted, &b.Failed, &b.Pending, &b.Unknown); err != nil {
			return page, err
		}
		page.Items = append(page.Items, b)
	}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		page.Next = page.Items[limit-1].ID
	}
	return page, rows.Err()
}

func (s *Store) Deliveries(session string, batch, after int64, limit int) (DeliveryPage, error) {
	limit = max(1, min(limit, 100))
	db, err := s.open()
	if err != nil {
		return DeliveryPage{}, err
	}
	defer func() { _ = db.Close() }()
	rows, err := db.Query(`SELECT d.id,d.payload,d.status,d.error FROM deliveries d JOIN batches b ON b.id=d.batch_id WHERE b.session=? AND b.id=? AND d.id>? ORDER BY d.id LIMIT ?`, session, batch, after, limit+1)
	if err != nil {
		return DeliveryPage{}, err
	}
	defer func() { _ = rows.Close() }()
	page := DeliveryPage{Items: []Delivery{}}
	for rows.Next() {
		var d Delivery
		var data []byte
		var status, e string
		if err := rows.Scan(&d.ID, &data, &status, &e); err != nil {
			return page, err
		}
		id := d.ID
		if err := json.Unmarshal(data, &d); err != nil {
			return page, err
		}
		d.ID = id
		d.Status = status
		d.Error = e
		page.Items = append(page.Items, d)
	}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		page.Next = page.Items[limit-1].ID
	}
	return page, rows.Err()
}

// Claim marks one message before network I/O. A restarted process must never
// automatically resend an ambiguous 'sending' delivery.
func (s *Store) Claim(id int64) (bool, error) {
	db, err := s.open()
	if err != nil {
		return false, err
	}
	defer func() { _ = db.Close() }()
	r, err := db.Exec("UPDATE deliveries SET status='sending',updated=? WHERE id=? AND status='pending'", time.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return false, err
	}
	n, err := r.RowsAffected()
	return n == 1, err
}

func (s *Store) Complete(id int64, sendError error) error {
	db, err := s.open()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	status, message := "accepted", ""
	if sendError != nil {
		status, message = "failed", sendError.Error()
		if errors.Is(sendError, ErrUncertain) {
			status = "unknown"
		}
	}
	_, err = db.Exec("UPDATE deliveries SET status=?,error=?,updated=? WHERE id=? AND status='sending'", status, message, time.Now().UTC().Format(time.RFC3339Nano), id)
	return err
}

func (s *Store) RecoverInterrupted() error {
	db, err := s.open()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	_, err = db.Exec("UPDATE deliveries SET status='unknown',error='Envio interrompido: confirme no provedor antes de enviar novamente' WHERE status='sending'")
	return err
}

func (s *Store) PendingBatches(session string) ([]int64, error) {
	db, err := s.open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = db.Close() }()
	rows, err := db.Query("SELECT DISTINCT b.id FROM batches b JOIN deliveries d ON d.batch_id=b.id WHERE b.session=? AND d.status='pending' ORDER BY b.id", session)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *Store) Retry(session string, batch int64, refresh ...func(Delivery) Delivery) (int64, error) {
	db, err := s.open()
	if err != nil {
		return 0, err
	}
	defer func() { _ = db.Close() }()
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	var repeated, pending int
	err = tx.QueryRow("SELECT repeated_by,(SELECT COUNT(*) FROM deliveries WHERE batch_id=batches.id AND status IN ('pending','sending')) FROM batches WHERE id=? AND session=?", batch, session).Scan(&repeated, &pending)
	if err != nil {
		return 0, err
	}
	if repeated != 0 || pending > 0 {
		return 0, errors.New("envio em andamento ou falhas já encaminhadas")
	}
	rows, err := tx.Query("SELECT payload FROM deliveries WHERE batch_id=? AND status='failed'", batch)
	if err != nil {
		return 0, err
	}
	items := []Delivery{}
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			_ = rows.Close()
			return 0, err
		}
		var d Delivery
		if err := json.Unmarshal(data, &d); err != nil {
			_ = rows.Close()
			return 0, err
		}
		d.Error = ""
		if len(refresh) > 0 {
			d = refresh[0](d)
		}
		items = append(items, d)
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return 0, err
	}
	if len(items) == 0 {
		return 0, errors.New("nenhuma falha confirmada para repetir")
	}
	id, err := insertBatch(tx, session, items)
	if err != nil {
		return 0, err
	}
	r, err := tx.Exec("UPDATE batches SET repeated_by=? WHERE id=? AND repeated_by=0", id, batch)
	if err != nil {
		return 0, err
	}
	n, err := r.RowsAffected()
	if err != nil || n != 1 {
		return 0, fmt.Errorf("falhas já encaminhadas")
	}
	return id, tx.Commit()
}
