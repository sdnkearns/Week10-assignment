package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"

	_ "github.com/marcboeker/go-duckdb"
)

const defaultDBPath = "./imdb.duckdb"

type Query struct {
	Title       string
	Description string
	SQL         string
	Columns     []string
}

var queries = []Query{
	{
		Title:       "1. Average Rank by Genre",
		Description: "Which genres receive the highest average movie ratings?",
		SQL: `
			SELECT
				g.genre,
				ROUND(AVG(m.rank), 3)  AS avg_rank,
				COUNT(DISTINCT m.movie_id) AS movie_count
			FROM movies m
			JOIN genres g ON m.movie_id = g.movie_id
			WHERE m.rank IS NOT NULL
			GROUP BY g.genre
			ORDER BY avg_rank DESC
			LIMIT 15`,
		Columns: []string{"Genre", "Avg Rank", "Movie Count"},
	},
	{
		Title:       "2. Most Prolific Actors",
		Description: "Which actors appeared in the most movies?",
		SQL: `
			SELECT
				a.first_name || ' ' || a.last_name AS actor,
				a.gender,
				COUNT(DISTINCT r.movie_id) AS movies_appeared_in
			FROM actors a
			JOIN roles r ON a.actor_id = r.actor_id
			GROUP BY a.actor_id, a.first_name, a.last_name, a.gender
			ORDER BY movies_appeared_in DESC
			LIMIT 10`,
		Columns: []string{"Actor", "Gender", "Movies"},
	},
	{
		Title:       "3. Genre Popularity Over Time",
		Description: "How many movies per genre were released each year? (top 5 genres, last 10 years in DB)",
		SQL: `
			WITH top_genres AS (
				SELECT genre
				FROM genres
				GROUP BY genre
				ORDER BY COUNT(*) DESC
				LIMIT 5
			),
			year_range AS (
				SELECT MAX(year) AS max_year FROM movies
			)
			SELECT
				m.year,
				g.genre,
				COUNT(DISTINCT m.movie_id) AS movies
			FROM movies m
			JOIN genres g ON m.movie_id = g.movie_id
			JOIN top_genres tg ON g.genre = tg.genre
			JOIN year_range yr ON m.year >= yr.max_year - 9
			WHERE m.year IS NOT NULL
			GROUP BY m.year, g.genre
			ORDER BY m.year DESC, movies DESC`,
		Columns: []string{"Year", "Genre", "Movies"},
	},
	{
		Title:       "4. Gender Breakdown by Genre",
		Description: "For each genre, what is the split of male vs. female actors?",
		SQL: `
			SELECT
				g.genre,
				COUNT(DISTINCT CASE WHEN a.gender = 'M' THEN a.actor_id END) AS male_actors,
				COUNT(DISTINCT CASE WHEN a.gender = 'F' THEN a.actor_id END) AS female_actors,
				COUNT(DISTINCT a.actor_id) AS total_actors
			FROM genres g
			JOIN roles r  ON g.movie_id  = r.movie_id
			JOIN actors a ON r.actor_id  = a.actor_id
			GROUP BY g.genre
			ORDER BY total_actors DESC
			LIMIT 12`,
		Columns: []string{"Genre", "Male", "Female", "Total"},
	},
}

func runQuery(db *sql.DB, q Query) error {
	fmt.Printf("%s\n", q.Title)
	fmt.Printf("%s\n", q.Description)

	rows, err := db.Query(q.SQL)
	if err != nil {
		return fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	colNames, err := rows.Columns()
	if err != nil {
		return err
	}
	if len(q.Columns) == len(colNames) {
		colNames = q.Columns
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

	fmt.Fprintln(w, strings.Join(colNames, "\t"))
	fmt.Fprintln(w, strings.Repeat("----------\t", len(colNames)))

	vals := make([]any, len(colNames))
	ptrs := make([]any, len(colNames))
	for i := range vals {
		ptrs[i] = &vals[i]
	}

	rowCount := 0
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return err
		}
		parts := make([]string, len(vals))
		for i, v := range vals {
			if v == nil {
				parts[i] = "NULL"
			} else {
				parts[i] = fmt.Sprintf("%v", v)
			}
		}
		fmt.Fprintln(w, strings.Join(parts, "\t"))
		rowCount++
	}
	w.Flush()

	fmt.Printf("  (%d rows)\n\n", rowCount)
	return rows.Err()
}

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = defaultDBPath
	}

	fmt.Printf("Database: %s\n", dbPath)

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("cannot connect to database: %v", err)
	}

	for _, q := range queries {
		if err := runQuery(db, q); err != nil {
			log.Printf("ERROR in %q: %v", q.Title, err)
		}
	}
}
