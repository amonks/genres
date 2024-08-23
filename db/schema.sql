-- Always use WAL mode so that we can support concurrent
-- connections to the database without corrupting data.
pragma journal_mode='wal';


-- GENRES
--

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

        fetched_artists_at datetime
);

create index if not exists genres_by_fetched_artists_at on genres ( fetched_artists_at );


-- ARTISTS
--

-- Artists holds the artists we've found using Spotify's search API.
create table if not exists artists (
        spotify_id text primary key,
        name       text,
        image_url  text,
        followers  integer,
        popularity integer,

        fetched_tracks_at       datetime,
        fetched_albums_at       datetime,
        indexed_genres_rtree_at datetime,
        indexed_tracks_rtree_at datetime
);

create index if not exists artists_by_fetched_tracks_at       on artists ( fetched_tracks_at );
create index if not exists artists_by_fetched_albums_at       on artists ( fetched_albums_at );
create index if not exists artists_by_indexed_genres_rtree_at on artists ( indexed_genres_rtree_at );
create index if not exists artists_by_indexed_tracks_rtree_at on artists ( indexed_tracks_rtree_at );


-- ALBUMS
--

create table if not exists albums (
        spotify_id   text primary key,
        name         text,

        -- Allowed values: "album", "single", "compilation"
        type         text,
        total_tracks integer,
        image_url    text,
        popularity   integer,

        -- Example: "1981-12"
        release_date           text,
        -- Allowed values: "year", "month", "day"
        release_date_precision text,

        fetched_tracks_at        datetime,
        indexed_tracks_rtree_at  datetime
);

create index if not exists albums_by_fetched_tracks_at on albums ( fetched_tracks_at );
create index if not exists indexed_tracks_rtree_at     on albums ( indexed_tracks_rtree_at );


-- TRACKS
--

create table if not exists tracks (
        spotify_id   text primary key,
        name         text,
        popularity   integer,

        album_spotify_id text,
        album_name       text,
        disc_number      integer,
        track_number     integer,

        fetched_analysis_at datetime,
        failed_analysis_at  datetime,
        indexed_search_at   datetime,

        key              integer,
        mode             integer,
        tempo            real,
        time_signature   integer,
        duration_ms      integer,

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

create view if not exists tracks_with_artist_names as
        select
                tracks.spotify_id as spotify_id,
                tracks.name as name,
                tracks.album_name as album_name,
                group_concat(artists.name, ' ') as artist_names,
                tracks.popularity as popularity
        from
                tracks
                        left join track_artists on tracks.spotify_id = track_artists.track_spotify_id
                        left join artists on track_artists.artist_spotify_id = artists.spotify_id
        group by tracks.spotify_id
        order by tracks.spotify_id asc;

create virtual table if not exists tracks_search using fts5(
        spotify_id,
        name,
        album_name,
        artist_names,
        popularity
);


-- ARTIST_GENRES
--

create table if not exists artist_genres (
        artist_spotify_id text references artists(spotify_id),
        genre_name        text references genres(name),

        primary key (artist_spotify_id, genre_name)
);

create index if not exists artist_genres_by_artist_spotify_id on artist_genres ( artist_spotify_id );
create index if not exists artist_genres_by_genre_name        on artist_genres ( genre_name );

create virtual table if not exists artist_genres_rtree using rtree(
        id,
        min_energy, max_energy,
        min_dynamic_variation, max_dynamic_variation,
        min_instrumentalness, max_instrumentalness,
        min_organicness, max_organicness,
        min_bounciness, max_bounciness
);


-- TRACK_ARTISTS
--

create table if not exists track_artists (
        track_spotify_id  text references tracks(spotify_id),
        artist_spotify_id text references artists(spotify_id),

        primary key (track_spotify_id, artist_spotify_id)
);

create index if not exists track_artists_by_track  on track_artists (track_spotify_id);
create index if not exists track_artists_by_artist on track_artists (artist_spotify_id);

create virtual table if not exists artist_tracks_rtree using rtree(
        id,
        min_energy, max_energy,
        min_dynamic_variation, max_dynamic_variation,
        min_instrumentalness, max_instrumentalness,
        min_organicness, max_organicness,
        min_bounciness, max_bounciness
);


-- ALBUM_ARTISTS
--

create table if not exists album_artists (
        artist_spotify_id text references artists(spotify_id),
        album_spotify_id  text references albums(spotify_id),

        primary key (artist_spotify_id, album_spotify_id)
);

create index if not exists album_artists_by_album  on album_artists (album_spotify_id);
create index if not exists album_artists_by_artist on album_artists (artist_spotify_id);


-- ALBUM_TRACKS
--

create table if not exists album_tracks (
        album_spotify_id text references albums(spotify_id),
        track_spotify_id text references tracks(spotify_id),

        primary key (album_spotify_id, track_spotify_id)
);

create index if not exists album_tracks_by_album  on album_tracks (album_spotify_id);
create index if not exists album_tracks_by_track  on album_tracks (track_spotify_id);

create virtual table if not exists album_tracks_rtree using rtree(
        id,
        min_energy, max_energy,
        min_dynamic_variation, max_dynamic_variation,
        min_instrumentalness, max_instrumentalness,
        min_organicness, max_organicness,
        min_bounciness, max_bounciness
);


-- ALBUM_GENRES
--

create table if not exists album_genres (
        album_spotify_id text references albums(spotify_id),
        genre_name text references genres(name),

        primary key (album_spotify_id, genre_name)
);

create index if not exists album_genres_by_album on album_genres ( album_spotify_id );
create index if not exists album_genres_by_genre on album_genres ( genre_name );
