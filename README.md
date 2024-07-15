This program tries to populate a sqlite3 database file with genres and artists
from Spotify and additional echonest-style metadata from everynoise.com.

See [db/schema.sql](https://github.com/amonks/genres/blob/main/db/schema.sql) for info about the resulting database.

See godoc at https://pkg.go.dev/github.com/amonks/genres for info about the code.

I'm running it now. it fetches all the genres very quickly from everynoise, but
with spotify's rate limiting, it looks like it'll take days/weeks to finish
populating artists

### genres

each genre has the following dimensions, which are all in the range [0, 4096]

- energy
- dynamic_variation
- instrumentalness
- organicness
- bounciness

so you can do a query to find some genres like this:

    select name
    from genres
    where energy between 1000 and 1500
    and instrumentalness between 800 and 900

### artists

each artist has a list of genres, from which we derive a _range_ on each of
those dimensions

like if bob dylan is folk and pop, he'll have a min_energy of min(folk.energy,
pop.energy)

you can think of an artist sort of like a bounding box drawn over part of the
visualization at everynoise.com, except the box isn't just xy, it's also the
three dimensions encoded by the colors the genre names are printed in

you can pick a point within that visualization and query to find all the artists
whose bounds contain that point

like the genre "klubowe" has,

- energy: 3390
- dynamic_variation: 2909
- instrumentalness: 1330
- organicness: 295
- bounciness: 2924

and if you want to find artists around that point, who may or may not be klubowe
artists, you can do a query like,

    select artists.name
    from artists join artists_rtree on artists.rowid = artists_rtree.id
    where 3390 between artists_rtree.min_energy and artists_rtree.max_energy
    and 2909 between artists_rtree.min_dynamic_variation and artists_rtree.max_dynamic_variation
    and 1330 between artists_rtree.min_instrumentalness and artists_rtree.max_instrumentalness
    and 295 between artists_rtree.min_organicness and artists_rtree.max_organicness
    and 2924 between artists_rtree.min_bounciness and artists_rtree.max_bounciness;
