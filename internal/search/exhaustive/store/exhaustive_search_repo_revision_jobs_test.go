package store_test

import (
	"context"
	"testing"

	"github.com/sourcegraph/log/logtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/search/exhaustive/store"
	"github.com/sourcegraph/sourcegraph/internal/search/exhaustive/types"
	"github.com/sourcegraph/sourcegraph/lib/errors"
)

func TestStore_CreateExhaustiveSearchRepoRevisionJob(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	logger := logtest.Scoped(t)
	db := database.NewDB(logger, dbtest.NewDB(logger, t))

	bs := basestore.NewWithHandle(db.Handle())

	t.Cleanup(func() {
		cleanupUsers(bs)
		cleanupRepos(bs)
		cleanupSearchJobs(bs)
		cleanupRepoJobs(bs)
	})

	userID, err := createUser(bs, "alice")
	require.NoError(t, err)
	repoID, err := createRepo(db, "repo-test")
	require.NoError(t, err)

	s := store.New(db, &observation.TestContext)

	searchJobID, err := s.CreateExhaustiveSearchJob(
		context.Background(),
		types.ExhaustiveSearchJob{InitiatorID: userID, Query: "repo:^github\\.com/hashicorp/errwrap$ hello"},
	)
	require.NoError(t, err)

	repoJobID, err := s.CreateExhaustiveSearchRepoJob(
		context.Background(),
		types.ExhaustiveSearchRepoJob{SearchJobID: searchJobID, RepoID: repoID},
	)
	require.NoError(t, err)

	tests := []struct {
		name        string
		setup       func(*testing.T, *store.Store)
		job         types.ExhaustiveSearchRepoRevisionJob
		expectedErr error
	}{
		{
			name: "New job",
			job: types.ExhaustiveSearchRepoRevisionJob{
				SearchRepoJobID: repoJobID,
				Revision:        "main",
			},
			expectedErr: nil,
		},
		{
			name: "Missing revision",
			job: types.ExhaustiveSearchRepoRevisionJob{
				SearchRepoJobID: repoJobID,
			},
			expectedErr: errors.New("missing revision"),
		},
		{
			name: "Missing repo job ID",
			job: types.ExhaustiveSearchRepoRevisionJob{
				Revision: "main",
			},
			expectedErr: errors.New("missing search repo job ID"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.setup != nil {
				test.setup(t, s)
			}

			jobID, err := s.CreateExhaustiveSearchRepoRevisionJob(context.Background(), test.job)

			if test.expectedErr != nil {
				require.Error(t, err)
				assert.EqualError(t, err, test.expectedErr.Error())
			} else {
				require.NoError(t, err)
				assert.NotZero(t, jobID)
			}
		})
	}
}