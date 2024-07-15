-- Always use WAL mode so that we can support concurrent
-- connections to the database without corrupting data.
pragma journal_mode='wal';

-- Genres holds the list of genres extracted from
-- everynoise.com.
--
-- Genres have many artists via the association table
-- artist_genres.
create table if not exists genres (
        name        text primary key,
        key         string,
        preview_url text,
        example     string,

        energy            integer,
        dynamic_variation integer,
        instrumentalness  integer,
        organicness       integer,
        bounciness        integer,

        popularity        integer
);

-- Artists holds the artists we've found using Spotify's search API. We try to
-- fetch the thousand first artists returned by a search for each genre.
--
-- Artists have many genres via the association table artist_genres.
--
-- Artists also have an rtree representing their bounds in genre-space. The 'id'
-- column in artists_rtree references the automatically-generated 'rowid' column
-- in artists.
create table if not exists artists (
        spotify_id text primary key,
        name       text,
        image_url  text,
        followers  integer,
        popularity integer
);

-- artists_rtree stores the bounds of each artist within 5-dimensional genre
-- space, and can be used for geospatial-style range queries.
--
-- For each artist, spotify gives us a list of genres. We can imagine finding
-- all of those genres on the everynoise website and drawing a bounding box
-- around them. Then, we can point to a spot on the visualization and query for
-- the artists whose boxes contain that point.
--
-- But actually: the visualization has more than two dimensions. In addition to
-- x and y position, each genre has a color, and the three channels of that
-- color (RGB) represent additional data.
create virtual table if not exists artists_rtree using rtree(
        id,
        min_energy, max_energy,
        min_dynamic_variation, max_dynamic_variation,
        min_instrumentalness, max_instrumentalness,
        min_organicness, max_organicness,
        min_bounciness, max_bounciness
);

-- artist_genres represents a many-to-many relationship between artists and
-- genres.
create table if not exists artist_genres (
        artist_spotify_id text references artists(spotify_id),
        genre_name        text references genres(name),

        primary key (artist_spotify_id, genre_name)
);

