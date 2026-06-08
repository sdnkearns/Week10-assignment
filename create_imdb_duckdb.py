import polars as pl
import duckdb
from pathlib import Path

CSV_DIR = Path(".")
DB_PATH = "imdb.duckdb"

TABLES: dict[str, dict] = {
    "movies": {
        "file": "aan520_w5a_movies2000After.csv",
        "schema": {
            "movie_id": pl.Int64,
            "name":     pl.Utf8,
            "year":     pl.Int32,
            "rank":     pl.Float32,
        },
    },
    "actors": {
        "file": "aan520_w5a_actors2000After.csv",
        "schema": {
            "actor_id":   pl.Int64,
            "first_name": pl.Utf8,
            "last_name":  pl.Utf8,
            "gender":     pl.Utf8,
        },
    },
    "genres": {
        "file": "aan520_w5a_genres2000After.csv",
        "schema": {
            "movie_id": pl.Int64,
            "genre":    pl.Utf8,
        },
    },
    "roles": {
        "file": "aan520_w5a_roles2000After.csv",
        "schema": {
            "actor_id": pl.Int64,
            "movie_id": pl.Int64,
            "role":     pl.Utf8,
        },
    },
}

def load_csv(path: Path, schema: dict) -> pl.DataFrame:
    print(f"  Reading {path.name} …", end=" ")
    df = pl.read_csv(
        path,
        schema_overrides=schema,
        null_values=["", "NULL", "null", "N/A", "\\N"],
        try_parse_dates=False,
        truncate_ragged_lines=True,
    )
    print(f"{df.shape[0]:,} rows × {df.shape[1]} cols")
    return df


def store_in_duckdb(conn: duckdb.DuckDBPyConnection, table: str, df: pl.DataFrame) -> None:
    conn.execute(f"DROP TABLE IF EXISTS {table}")
    conn.execute(f"CREATE TABLE {table} AS SELECT * FROM df")
    row_count = conn.execute(f"SELECT COUNT(*) FROM {table}").fetchone()[0]
    print(f"Table '{table}' stored — {row_count:,} rows")

def main() -> None:

    print("Loading CSV files:")
    dataframes: dict[str, pl.DataFrame] = {}
    for table_name, cfg in TABLES.items():
        csv_path = CSV_DIR / cfg["file"]
        if not csv_path.exists():
            raise FileNotFoundError(
                f"File not found: {csv_path}\n"
            )
        dataframes[table_name] = load_csv(csv_path, cfg["schema"])

    print()


    # 3. Write to DuckDB
    print(f"Writing to DuckDB database: {DB_PATH}")
    with duckdb.connect(DB_PATH) as conn:
        for table_name, df in dataframes.items():
            store_in_duckdb(conn, table_name, df)

    print(f"\nDatabase saved to: {Path(DB_PATH).resolve()}")


if __name__ == "__main__":
    main()