This program tries to download track analysis for every track on Spotify,
populating a sqlite database (./genres.db, by default), and presents an
interface for querying that downloaded data. At Spotify's default rate limit of
~6,000 requests per day, the download will take years to complete.

See [db/schema.sql](https://github.com/amonks/genres/blob/main/db/schema.sql)
for info about the resulting database.

See godoc at https://pkg.go.dev/github.com/amonks/genres for info about the code.

See `genres -help` for info about CLI commands and flags.
