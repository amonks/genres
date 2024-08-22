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
        example     string,

        energy            real,
        dynamic_variation real,
        instrumentalness  real,
        organicness       real,
        bounciness        real,

        popularity        real,

        fetched_artists_at text
);

create index if not exists genres_by_fetched_artists_at on genres ( fetched_artists_at );

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

        fetched_tracks_at text,
        fetched_albums_at text
);

create index if not exists artists_by_fetched_tracks_at on artists ( fetched_tracks_at );
create index if not exists artists_by_fetched_albums_at on artists ( fetched_albums_at );

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

        fetched_tracks_at text
);

create index if not exists albums_by_fetched_tracks_at on albums ( fetched_tracks_at );

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
        spotify_id   text primary key,
        name         text,
        popularity   integer,

        album_spotify_id text,
        album_name       text,
        disc_number      integer,
        track_number     integer,

        fetched_analysis_at text,
        failed_analysis_at text,
        indexed_search_at text,

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

create index if not exists tracks_by_fetched_analysis_at on tracks ( fetched_analysis_at );
create index if not exists tracks_by_failed_analysis_at  on tracks ( failed_analysis_at );
create index if not exists tracks_by_indexed_search_at   on tracks ( indexed_search_at );

create index if not exists tracks_by_key               on tracks ( key              );
create index if not exists tracks_by_mode              on tracks ( mode             );
create index if not exists tracks_by_tempo             on tracks ( tempo            );
create index if not exists tracks_by_time_signature    on tracks ( time_signature   );
create index if not exists tracks_by_popularity        on tracks ( popularity       );

create index if not exists tracks_by_acousticness      on tracks ( acousticness     );
create index if not exists tracks_by_danceability      on tracks ( danceability     );
create index if not exists tracks_by_energy            on tracks ( energy           );
create index if not exists tracks_by_instrumentalness  on tracks ( instrumentalness );
create index if not exists tracks_by_liveness          on tracks ( liveness         );
create index if not exists tracks_by_loudness          on tracks ( loudness         );
create index if not exists tracks_by_speechiness       on tracks ( speechiness      );
create index if not exists tracks_by_valence           on tracks ( valence          );

create table if not exists track_artists (
        track_spotify_id  text references tracks(spotify_id),
        artist_spotify_id text references artists(spotify_id),

        primary key (track_spotify_id, artist_spotify_id)
);

create index if not exists track_artists_by_track  on track_artists (track_spotify_id);
create index if not exists track_artists_by_artist on track_artists (artist_spotify_id);

create table if not exists album_artists (
        artist_spotify_id text references artists(spotify_id),
        album_spotify_id  text references albums(spotify_id),

        primary key (artist_spotify_id, album_spotify_id)
);

create index if not exists album_artists_by_album  on album_artists (album_spotify_id);
create index if not exists album_artists_by_artist on album_artists (artist_spotify_id);

create table if not exists album_tracks (
        album_spotify_id text references albums(spotify_id),
        track_spotify_id text references tracks(spotify_id),

        primary key (album_spotify_id, track_spotify_id)
);

create index if not exists album_tracks_by_album  on album_tracks (album_spotify_id);
create index if not exists album_tracks_by_track  on album_tracks (track_spotify_id);

create virtual table if not exists tracks_search using fts5(
        track_spotify_id,
        content,
);
