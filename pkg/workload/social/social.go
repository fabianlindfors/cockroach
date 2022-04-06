package social

import (
	"github.com/cockroachdb/cockroach/pkg/workload"
	"github.com/cockroachdb/cockroach/pkg/workload/histogram"
	"github.com/cockroachdb/cockroach/pkg/util/timeutil"
	"github.com/spf13/pflag"
	"math/rand"
	"context"
	gosql "database/sql"
	"strings"
)

type social struct {
	flags     workload.Flags
	connFlags *workload.ConnFlags

	postsCount int
	likesCount int

	readPercentage int
	splits      int
}

func init() {
	workload.Register(socialMeta)
}

var socialMeta = workload.Meta{
	Name:         `social`,
	Description:  `Benchmark for testing uncertainty restarts`,
	Version:      `1.0.0`,
	PublicFacing: true,
	New: func() workload.Generator {
		g := &social{}
		g.flags.FlagSet = pflag.NewFlagSet(`social`, pflag.ContinueOnError)
		g.flags.Meta = map[string]workload.FlagMeta{
			`workload`: {RuntimeOnly: true},
		}

		g.flags.IntVar(&g.postsCount, `posts`, 100, `Number of posts to use`)
		g.flags.IntVar(&g.likesCount, `likes`, 10000, `Number of likes to add initially`)
		g.flags.IntVar(&g.readPercentage, `read-percentage`, 95, `Percentage of operations which should be reads`)
		g.flags.IntVar(&g.splits, `splits`, 0, `Number of splits to perform before starting normal operations`)

		g.connFlags = workload.NewConnFlags(&g.flags)
		return g
	},
}

func (*social) Meta() workload.Meta { return socialMeta }

// Flags implements the Flagser interface.
func (g *social) Flags() workload.Flags { return g.flags }

// Hooks implements the Hookser interface.
func (g *social) Hooks() workload.Hooks {
	return workload.Hooks{}
}

// Tables implements the Generator interface.
func (g *social) Tables() []workload.Table {
	postsTable := workload.Table{
		Name: `posts`,
		Schema: "(id INTEGER PRIMARY KEY, title TEXT)",
		InitialRows: workload.Tuples(
			g.postsCount,
			func(rowIdx int) []interface{} {
				return []interface{}{rowIdx, randString()}
			},
		),
		Splits: workload.Tuples(
			g.splits,
			func(splitIdx int) []interface{} {
				step := g.postsCount / (g.splits+1)
				return []interface{}{step*(splitIdx+1)}
			},
		),
	}

	likesTable := workload.Table{
		Name: `likes`,
		Schema: `(
			id SERIAL PRIMARY KEY,
			post_id INTEGER,
			FOREIGN KEY (post_id) REFERENCES posts (id),
			INDEX (post_id)
		)`,
		InitialRows: workload.Tuples(
			g.likesCount,
			func(rowIdx int) []interface{} {
				postId := rand.Intn(g.postsCount)
				return []interface{}{rowIdx, postId}
			},
		),
		Splits: workload.Tuples(
			g.splits,
			func(splitIdx int) []interface{} {
				step := g.likesCount / (g.splits+1)
				return []interface{}{step*(splitIdx+1)}
			},
		),
	}

	return []workload.Table{postsTable, likesTable}
}

func randString() string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

    b := make([]rune, 10)
    for i := range b {
        b[i] = letterRunes[rand.Intn(len(letterRunes))]
    }
    return string(b)
}

// Ops implements the Opser interface.
func (g *social) Ops(
	ctx context.Context, urls []string, reg *histogram.Registry,
) (workload.QueryLoad, error) {
	sqlDatabase, err := workload.SanitizeUrls(g, g.connFlags.DBOverride, urls)
	if err != nil {
		return workload.QueryLoad{}, err
	}
	db, err := gosql.Open(`cockroach`, strings.Join(urls, ` `))
	if err != nil {
		return workload.QueryLoad{}, err
	}
	// Allow a maximum of concurrency+1 connections to the database.
	db.SetMaxOpenConns(g.connFlags.Concurrency + 1)
	db.SetMaxIdleConns(g.connFlags.Concurrency + 1)

	readStmt, err := db.Prepare(`
		SELECT posts.id, posts.title, COUNT(*)
		FROM posts
		LEFT JOIN likes ON posts.id = likes.post_id
		GROUP BY posts.id
		ORDER BY COUNT(*) DESC
	`)
	if err != nil {
		return workload.QueryLoad{}, err
	}

	likeStmt, err := db.Prepare(`
		INSERT INTO likes(post_id)
		VALUES ($1)
	`)
	if err != nil {
		return workload.QueryLoad{}, err
	}

	ql := workload.QueryLoad{SQLDatabase: sqlDatabase}
	for i := 0; i < g.connFlags.Concurrency; i++ {
		rng := rand.New(rand.NewSource(timeutil.Now().UnixNano()))
		w := &socialWorker{
			config:                  g,
			hists:                   reg.GetHandle(),
			readStmt:                readStmt,
			likeStmt: 				 likeStmt,
			rng:                     rng,
		}
		ql.WorkerFns = append(ql.WorkerFns, w.run)
	}
	return ql, nil
}

type operation string

const (
	readOp operation = "read"
	likeOp operation = "like"
)

type socialWorker struct {
	config *social
	hists  *histogram.Histograms
	db     *gosql.DB

	readStmt *gosql.Stmt
	likeStmt *gosql.Stmt

	rng           *rand.Rand    // used to generate random strings for the values
}

func (sw *socialWorker) run(ctx context.Context) error {
	op := sw.getOp()
	var err error

	start := timeutil.Now()

	switch op {
	case readOp:
		err = sw.read(ctx)
	case likeOp:
		err = sw.like(ctx)
	}
	if err != nil {
		return err
	}

	elapsed := timeutil.Since(start)
	sw.hists.Get(string(op)).Record(elapsed)

	return nil
}

func (sw *socialWorker) read(ctx context.Context) error {
	res, err := sw.readStmt.QueryContext(ctx)
	defer res.Close()
	return err
}

func (sw *socialWorker) like(ctx context.Context) error {
	postToLike := rand.Intn(sw.config.postsCount)
	_, err := sw.likeStmt.ExecContext(ctx, postToLike)
	return err
}

func (sw *socialWorker) getOp() operation {
	if sw.rng.Float64() < float64(sw.config.readPercentage)/100.0 {
		return readOp
	} else {
		return likeOp
	}
}