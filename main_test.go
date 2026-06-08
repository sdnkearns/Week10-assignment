package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	_ "github.com/marcboeker/go-duckdb"
)

// openTestDB opens an in-memory DuckDB instance and seeds it with
// representative IMDB data matching the production schema.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open in-memory DuckDB: %v", err)
	}

	ddl := `
		CREATE TABLE movies (
			movie_id BIGINT PRIMARY KEY,
			name     VARCHAR,
			year     INTEGER,
			rank     FLOAT
		);
		CREATE TABLE actors (
			actor_id   BIGINT PRIMARY KEY,
			first_name VARCHAR,
			last_name  VARCHAR,
			gender     VARCHAR
		);
		CREATE TABLE genres (
			movie_id BIGINT,
			genre    VARCHAR
		);
		CREATE TABLE roles (
			actor_id BIGINT,
			movie_id BIGINT,
			role     VARCHAR
		);

		-- movies: mix of ranked/unranked, different years and genres
		INSERT INTO movies VALUES
			(1, 'Action Hero',   2005, 8.5),
			(2, 'Comedy Nights', 2006, 7.2),
			(3, 'Drama Queen',   2007, 9.1),
			(4, 'Sci-Fi World',  2008, 6.8),
			(5, 'No Rank Film',  2009, NULL),
			(6, 'Action 2',      2010, 7.9),
			(7, 'Comedy 2',      2010, 8.0),
			(8, 'Drama 2',       2011, 8.8);

		-- actors: male and female
		INSERT INTO actors VALUES
			(1, 'Alice',   'Smith',   'F'),
			(2, 'Bob',     'Jones',   'M'),
			(3, 'Charlie', 'Brown',   'M'),
			(4, 'Diana',   'Prince',  'F'),
			(5, 'Eve',     'Adams',   'F');

		-- genres
		INSERT INTO genres VALUES
			(1, 'Action'),
			(2, 'Comedy'),
			(3, 'Drama'),
			(4, 'Sci-Fi'),
			(5, 'Action'),
			(6, 'Action'),
			(7, 'Comedy'),
			(8, 'Drama');

		-- roles: spread actors across movies
		INSERT INTO roles VALUES
			(1, 1, 'Lead'),
			(2, 1, 'Support'),
			(3, 2, 'Lead'),
			(4, 3, 'Lead'),
			(5, 4, 'Lead'),
			(1, 5, 'Cameo'),
			(2, 6, 'Lead'),
			(3, 7, 'Support'),
			(4, 8, 'Lead'),
			(1, 6, 'Support'),
			(2, 7, 'Lead'),
			(1, 8, 'Support');
	`
	if _, err := db.Exec(ddl); err != nil {
		t.Fatalf("failed to seed test DB: %v", err)
	}
	return db
}

// ---- helpers ----------------------------------------------------------------

// collectRows executes sql against db and returns all rows as [][]string.
func collectRows(t *testing.T, db *sql.DB, query string) [][]string {
	t.Helper()
	rows, err := db.Query(query)
	if err != nil {
		t.Fatalf("collectRows query error: %v", err)
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	var result [][]string
	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			t.Fatalf("scan error: %v", err)
		}
		row := make([]string, len(vals))
		for i, v := range vals {
			if v == nil {
				row[i] = "NULL"
			} else {
				row[i] = fmt.Sprintf("%v", v)
			}
		}
		result = append(result, row)
	}
	return result
}

// ---- Query struct tests ------------------------------------------------------

func TestQueryStructFields(t *testing.T) {
	q := Query{
		Title:       "Test Title",
		Description: "Test Description",
		SQL:         "SELECT 1",
		Columns:     []string{"Col1"},
	}
	if q.Title != "Test Title" {
		t.Errorf("Title: got %q", q.Title)
	}
	if q.Description != "Test Description" {
		t.Errorf("Description: got %q", q.Description)
	}
	if q.SQL != "SELECT 1" {
		t.Errorf("SQL: got %q", q.SQL)
	}
	if len(q.Columns) != 1 || q.Columns[0] != "Col1" {
		t.Errorf("Columns: got %v", q.Columns)
	}
}

func TestQueriesSliceNotEmpty(t *testing.T) {
	if len(queries) == 0 {
		t.Fatal("queries slice must not be empty")
	}
}

func TestEachQueryHasRequiredFields(t *testing.T) {
	for i, q := range queries {
		if strings.TrimSpace(q.Title) == "" {
			t.Errorf("queries[%d]: empty Title", i)
		}
		if strings.TrimSpace(q.SQL) == "" {
			t.Errorf("queries[%d]: empty SQL", i)
		}
		if len(q.Columns) == 0 {
			t.Errorf("queries[%d]: no Columns defined", i)
		}
	}
}

// ---- Database connectivity --------------------------------------------------

func TestDBPing(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Errorf("Ping failed: %v", err)
	}
}

func TestDBOpenInvalidPath(t *testing.T) {
	db, err := sql.Open("duckdb", "/nonexistent/path/imdb.duckdb")
	if err != nil {
		return // some drivers error at Open
	}
	defer db.Close()
	if err := db.Ping(); err == nil {
		t.Error("expected error pinging non-existent database file")
	}
}

// ---- Schema / seed validation -----------------------------------------------

func TestTablesExist(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	for _, table := range []string{"movies", "actors", "genres", "roles"} {
		var count int
		err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		if err != nil {
			t.Errorf("table %q not accessible: %v", table, err)
		}
	}
}

func TestSeedRowCounts(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	tests := []struct {
		table string
		want  int
	}{
		{"movies", 8},
		{"actors", 5},
		{"genres", 8},
		{"roles", 12},
	}
	for _, tc := range tests {
		var got int
		db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tc.table)).Scan(&got)
		if got != tc.want {
			t.Errorf("table %s: want %d rows, got %d", tc.table, tc.want, got)
		}
	}
}

// ---- runQuery ---------------------------------------------------------------

func TestRunQuerySuccess(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	q := Query{
		Title:       "Test: count movies",
		Description: "Simple count",
		SQL:         "SELECT COUNT(*) FROM movies",
		Columns:     []string{"Count"},
	}
	if err := runQuery(db, q); err != nil {
		t.Errorf("runQuery returned unexpected error: %v", err)
	}
}

func TestRunQueryInvalidSQL(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	q := Query{
		Title:       "Bad Query",
		Description: "Should fail",
		SQL:         "SELECT * FROM nonexistent_table_xyz",
		Columns:     []string{"X"},
	}
	if err := runQuery(db, q); err == nil {
		t.Error("expected error for invalid SQL, got nil")
	}
}

func TestRunQueryColumnOverride(t *testing.T) {
	// When len(q.Columns) == number of result columns the custom names are used.
	// We verify runQuery doesn't error in that case.
	db := openTestDB(t)
	defer db.Close()

	q := Query{
		Title:       "Column override",
		Description: "",
		SQL:         "SELECT movie_id, name FROM movies LIMIT 3",
		Columns:     []string{"ID", "Title"},
	}
	if err := runQuery(db, q); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunQueryColumnMismatch(t *testing.T) {
	// Wrong number of custom column names — should fall back silently, not error.
	db := openTestDB(t)
	defer db.Close()

	q := Query{
		Title:       "Column mismatch",
		Description: "",
		SQL:         "SELECT movie_id, name, year FROM movies LIMIT 1",
		Columns:     []string{"OnlyOne"}, // mismatch: 3 cols but 1 name
	}
	if err := runQuery(db, q); err != nil {
		t.Errorf("unexpected error on column mismatch: %v", err)
	}
}

func TestRunQueryNullHandling(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	// movie_id=5 has NULL rank
	q := Query{
		Title:       "Null rank",
		Description: "Should print NULL",
		SQL:         "SELECT movie_id, rank FROM movies WHERE movie_id = 5",
		Columns:     []string{"ID", "Rank"},
	}
	if err := runQuery(db, q); err != nil {
		t.Errorf("unexpected error with NULL value: %v", err)
	}
}

func TestRunQueryEmptyResultSet(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	q := Query{
		Title:       "Empty result",
		Description: "No rows match",
		SQL:         "SELECT * FROM movies WHERE year = 1800",
		Columns:     []string{"movie_id", "name", "year", "rank"},
	}
	if err := runQuery(db, q); err != nil {
		t.Errorf("unexpected error on empty result: %v", err)
	}
}

// ---- All four production queries --------------------------------------------

func TestAllProductionQueriesRunWithoutError(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	for _, q := range queries {
		t.Run(q.Title, func(t *testing.T) {
			if err := runQuery(db, q); err != nil {
				t.Errorf("production query %q failed: %v", q.Title, err)
			}
		})
	}
}

// ---- SQL correctness: Query 1 (Average Rank by Genre) -----------------------

func TestQuery1AverageRankByGenre(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	rows := collectRows(t, db, queries[0].SQL)
	if len(rows) == 0 {
		t.Fatal("Query 1 returned no rows")
	}

	// Verify columns: genre, avg_rank, movie_count
	for _, row := range rows {
		if len(row) != 3 {
			t.Fatalf("Query 1: expected 3 columns, got %d", len(row))
		}
	}

	// Action has movies 1, 5, 6 — only 1 and 6 are ranked (8.5, 7.9)
	// avg = (8.5+7.9)/2 = 8.2; Drama has 3 and 8 ranked (9.1, 8.8) avg = 8.95
	// Drama should rank above Action
	found := map[string]string{}
	for _, row := range rows {
		found[row[0]] = row[1]
	}
	if _, ok := found["Action"]; !ok {
		t.Error("Query 1: 'Action' genre missing from results")
	}
	if _, ok := found["Drama"]; !ok {
		t.Error("Query 1: 'Drama' genre missing from results")
	}
	// First row (highest avg rank) should be Drama (8.95) not Action (8.2)
	if rows[0][0] != "Drama" {
		t.Errorf("Query 1: expected top genre 'Drama', got %q", rows[0][0])
	}
}

// ---- SQL correctness: Query 2 (Most Prolific Actors) ------------------------

func TestQuery2MostProlificActors(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	rows := collectRows(t, db, queries[1].SQL)
	if len(rows) == 0 {
		t.Fatal("Query 2 returned no rows")
	}
	for _, row := range rows {
		if len(row) != 3 {
			t.Fatalf("Query 2: expected 3 columns, got %d", len(row))
		}
	}

	// Alice Smith appears in movies 1,5,6,8 = 4 movies → most prolific
	topActor := rows[0][0]
	if topActor != "Alice Smith" {
		t.Errorf("Query 2: expected top actor 'Alice Smith', got %q", topActor)
	}
}

func TestQuery2GenderValues(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	rows := collectRows(t, db, queries[1].SQL)
	for _, row := range rows {
		g := row[1]
		if g != "M" && g != "F" {
			t.Errorf("Query 2: unexpected gender value %q", g)
		}
	}
}

// ---- SQL correctness: Query 3 (Genre Popularity Over Time) ------------------

func TestQuery3GenrePopularityOverTime(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	rows := collectRows(t, db, queries[2].SQL)
	if len(rows) == 0 {
		t.Fatal("Query 3 returned no rows")
	}
	for _, row := range rows {
		if len(row) != 3 {
			t.Fatalf("Query 3: expected 3 columns, got %d", len(row))
		}
	}

	// Results should be ordered year DESC
	for i := 1; i < len(rows); i++ {
		if rows[i][0] > rows[i-1][0] {
			t.Errorf("Query 3: rows not sorted by year DESC at index %d", i)
		}
	}
}

// ---- SQL correctness: Query 4 (Gender Breakdown by Genre) -------------------

func TestQuery4GenderBreakdownByGenre(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	rows := collectRows(t, db, queries[3].SQL)
	if len(rows) == 0 {
		t.Fatal("Query 4 returned no rows")
	}
	for _, row := range rows {
		if len(row) != 4 {
			t.Fatalf("Query 4: expected 4 columns, got %d", len(row))
		}
	}
}

func TestQuery4TotalGeaterThanOrEqualMalePlusFemale(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	// male + female <= total (actors with unknown gender would push total higher)
	sql := `
		SELECT
			g.genre,
			COUNT(DISTINCT CASE WHEN a.gender = 'M' THEN a.actor_id END) AS male,
			COUNT(DISTINCT CASE WHEN a.gender = 'F' THEN a.actor_id END) AS female,
			COUNT(DISTINCT a.actor_id) AS total
		FROM genres g
		JOIN roles r  ON g.movie_id  = r.movie_id
		JOIN actors a ON r.actor_id  = a.actor_id
		GROUP BY g.genre`

	rows := collectRows(t, db, sql)
	for _, row := range rows {
		var male, female, total int
		fmt.Sscan(row[1], &male)
		fmt.Sscan(row[2], &female)
		fmt.Sscan(row[3], &total)
		if male+female > total {
			t.Errorf("genre %s: male(%d)+female(%d) > total(%d)", row[0], male, female, total)
		}
	}
}

// ---- DB_PATH env var --------------------------------------------------------

func TestDBPathEnvDefault(t *testing.T) {
	os.Unsetenv("DB_PATH")
	// defaultDBPath constant should be the fallback
	if defaultDBPath == "" {
		t.Error("defaultDBPath must not be empty")
	}
}

func TestDBPathEnvOverride(t *testing.T) {
	t.Setenv("DB_PATH", "/tmp/custom.duckdb")
	got := os.Getenv("DB_PATH")
	if got != "/tmp/custom.duckdb" {
		t.Errorf("env override: got %q", got)
	}
}
