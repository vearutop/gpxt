package runnerup

import (
	"context"
	"time"

	"github.com/bool64/sqluct"
	_ "modernc.org/sqlite"
)

// ActivityType describes what was happening during tracking.
type ActivityType int

// MSTimeStamp is a UNIX timestamp in milliseconds.
type MSTimeStamp int

func (t MSTimeStamp) String() string {
	return t.Time().Format(time.RFC3339)
}

func (t MSTimeStamp) Time() time.Time {
	return time.Unix(int64(t/1000), 0)
}

// TimeStamp is a UNIX timestamp in seconds.
type TimeStamp int

func (t TimeStamp) String() string {
	return time.Unix(int64(t), 0).Format(time.RFC3339)
}

type Seconds int

func (t Seconds) String() string {
	return (time.Duration(t) * time.Second).String()
}

// Activity is a DB row that may own multiple items of Location..
type Activity struct {
	ID        int          `db:"_id"`
	StartTime TimeStamp    `db:"start_time"`
	Distance  float64      `db:"distance"`
	Time      Seconds      `db:"time"`
	Type      ActivityType `db:"type"`
}

// Location is a DB row that holds GPS point, owned by Activity.
type Location struct {
	ID int `db:"_id"`

	ActivityID int         `db:"activity_id"`
	Time       MSTimeStamp `db:"time"`
	Lon        float64     `db:"longitude"`
	Lat        float64     `db:"latitude"`
	Alt        float64     `db:"altitude"`

	Accuracy   float64  `db:"accurancy"`
	Speed      float64  `db:"speed"`
	Bearing    *float64 `db:"bearing"`
	Satellites int      `db:"satellites"`
}

type Repository struct {
	st *sqluct.Storage
	ls sqluct.StorageOf[Location]
	as sqluct.StorageOf[Activity]
}

func NewRepository(dbFile string) (*Repository, error) {
	st, err := sqluct.Open("sqlite", dbFile)
	if err != nil {
		return nil, err
	}

	r := Repository{}
	r.st = st
	r.ls = sqluct.Table[Location](st, "location")
	r.as = sqluct.Table[Activity](st, "activity")

	return &r, nil
}

func (r *Repository) Close() error {
	return r.st.DB().Close()
}

func (r *Repository) ListActivities(ctx context.Context, limit uint64) ([]Activity, error) {
	return r.as.List(ctx, r.as.SelectStmt().
		OrderBy(r.as.Fmt("%s DESC", &r.as.R.ID)).
		Limit(limit))
}

func (r *Repository) ListLocations(ctx context.Context, activityID int) ([]Location, error) {
	return r.ls.List(ctx, r.ls.SelectStmt().
		OrderBy(r.ls.Fmt("%s ASC", &r.ls.R.ID)).
		Where(r.ls.Eq(&r.ls.R.ActivityID, activityID)),
	)
}
