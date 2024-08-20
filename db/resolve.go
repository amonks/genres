package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/amonks/genres/data"
)

func (db *DB) Resolve(ctx context.Context, input string) (*data.Track, error) {
	parts := strings.SplitN(input, ":", 2)
	if len(parts) == 1 {
		return db.GetTrack(ctx, input)
	}
	cmd, arg := parts[0], parts[1]
	switch cmd {
	case "q":
		tracks, err := db.Search(ctx, arg, 1)
		if err != nil {
			return nil, err
		}
		if len(tracks) == 0 {
			return nil, fmt.Errorf("no track found for query '%s'", arg)
		}
		return &tracks[0], nil
	case "id":
		return db.GetTrack(ctx, arg)
	default:
		return nil, fmt.Errorf("unknown search cmd '%s'", cmd)
	}
}
