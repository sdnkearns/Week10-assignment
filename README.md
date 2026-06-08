# Week10-assignment

Week 10 assignment for MSDS 431, Building a Personal Movie Database

data input files:  
  aan520_w5a_actors2000After.csv  
  aan520_w5a_genres2000After.csv  
  aan520_w5a_movies2000After.csv  
  aan520_w5a_roles2000After.csv  

create_imdb_duckdb.py  
  -python script that reads the input data csv files, puts the data into a polars dataframe, then stores that data in a DuckDB database

imdb.duckdb  
  -DuckDB database created by the python script

go.mod, go.sum, imdb_query.go  
  -go project files for accessing the imdb DuckDB database, and executing some example queries

main_test.go  
  -contains unit tests for imdb_query.go

create duckdb database as:  
  python create_imdb_database.py

run go query as:  
  go run imdb_query.go

build go query as:  
  go build imdb_query.go

run unit tests on go query script:  
  go test -v

To set up the SQLite relational database, I created the DB schema with a python dictionary based on the input tables, and used that as the schema with the polars read_csv function to create my dataframe. I then created a new DuckDB database, and looped through all the tables in the dataframe to store the information in the DB. To add a new table with movies in my personal collection, first I would need to create a new csv file containing the movie_id, the title of the movie, and my personal rating. I would need to find the IMDB movie_id for each movie, due to movie_id being the primary key, in the movies table, and it is possible for different movies to have the same title, but they will not have the same movie_id. This information would be entered into the DB the same way as the other tables were, and I would need to add it to my schema in the python code. This table could be useful if I wanted to compare my scores with the average review scores on IMDB, or if I wanted to find new movies with the same genre or actors. With this, you can get personalized recommendations for movies similar to ones you like. 

To further enhance the database, you could add new tables with additional information about each film, such as studio, director, producer, budget, etc. You could also add tables with information to help personalize searches, like common tropes and themes found in each movie, ratings/content warnings, or lists of other movies in the same franchise. 

I used claude to generate unit tests for imdb_query.go, the transcript can be found in claude_transcript.log
