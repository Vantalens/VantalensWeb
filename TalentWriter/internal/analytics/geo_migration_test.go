package analytics

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestInitMigratesLegacyGeoCacheCoordinates(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy-visits.db")
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`CREATE TABLE geo_cache (
		ip TEXT PRIMARY KEY,
		country TEXT NOT NULL,
		region TEXT NOT NULL,
		city TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`); err != nil {
		t.Fatal(err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}

	t.Setenv("ANALYTICS_DB_PATH", dbPath)
	if err := Init(filepath.Dir(dbPath)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = Close() })

	conn, err = getDB()
	if err != nil {
		t.Fatal(err)
	}
	rows, err := conn.Query(`PRAGMA table_info(geo_cache)`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	columns := map[string]bool{}
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatal(err)
		}
		columns[name] = true
	}
	if !columns["latitude"] || !columns["longitude"] {
		t.Fatalf("migrated geo_cache columns = %v", columns)
	}
}
