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

        popularity        integer,

        has_fetched_artists boolean not null default false
);

create index if not exists genres_by_has_fetched_artists on genres ( has_fetched_artists );

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
        popularity integer,

        has_fetched_tracks boolean not null default false,
        has_fetched_albums boolean not null default false
);

create index if not exists artists_by_has_fetched_tracks on artists ( has_fetched_tracks );
create index if not exists artists_by_has_fetched_albums on artists ( has_fetched_albums );

create table if not exists albums (
        spotify_id   text primary key,
        name         text,

        -- Allowed values: "album", "single", "compilation"
        type         text,
        total_tracks integer,
        image_url    text,

        -- Example: "1981-12"
        release_date           text,
        -- Allowed values: "year", "month", "day"
        release_date_precision text,

        has_fetched_tracks boolean not null default false
);

create index if not exists albums_by_has_fetched_tracks on albums ( has_fetched_tracks );

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

create table if not exists tracks (
        spotify_id   text primary_key,
        name         text,
        preview_url  text,
        duration_ms  integer,
        popularity   integer,

        album_spotify_id text,
        album_name       text,
        disc_number      integer,
        track_number     integer,

        has_analysis     boolean not null default false,

        key              integer,
        mode             integer,
        tempo            real,
        time_signature   integer,

        acousticness     real,
        danceability     real,
        energy           real,
        instrumentalness real,
        liveness         real,
        loudness         real,
        speechiness      real,
        valence          real
);

create index if not exists tracks_by_has_analysis on tracks ( has_analysis );

create table if not exists track_artists (
        track_spotify_id  text references tracks(spotify_id),
        artist_spotify_id text references artists(spotify_id),

        primary key (track_spotify_id, artist_spotify_id)
);

create table if not exists album_artists (
        artist_spotify_id text references artists(spotify_id),
        album_spotify_id  text references albums(spotify_id),

        primary key (artist_spotify_id, album_spotify_id)
);

create table if not exists album_tracks (
        album_spotify_id text references albums(spotify_id),
        track_spotify_id text references tracks(spotify_id),

        primary key (album_spotify_id, track_spotify_id)
);
