package store_test

import (
	"context"

	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

func createUser(store *basestore.Store, username string) (int32, error) {
	q := sqlf.Sprintf(`INSERT INTO users(username) VALUES(%s) RETURNING id`, username)
	row := store.QueryRow(context.Background(), q)
	var userID int32
	err := row.Scan(&userID)
	return userID, err
}

func cleanupUsers(store *basestore.Store) error {
	q := sqlf.Sprintf(`TRUNCATE TABLE users RESTART IDENTITY CASCADE`)
	return store.Exec(context.Background(), q)
}

func createRepo(db database.DB, name string) (api.RepoID, error) {
	repoStore := db.Repos()
	repo := types.Repo{Name: api.RepoName(name)}
	err := repoStore.Create(context.Background(), &repo)
	return repo.ID, err
}

func cleanupRepos(store *basestore.Store) error {
	q := sqlf.Sprintf(`TRUNCATE TABLE repo`)
	return store.Exec(context.Background(), q)
}

func cleanupSearchJobs(store *basestore.Store) error {
	q := sqlf.Sprintf(`TRUNCATE TABLE exhaustive_search_jobs`)
	return store.Exec(context.Background(), q)
}

func cleanupRepoJobs(store *basestore.Store) error {
	q := sqlf.Sprintf(`TRUNCATE TABLE exhaustive_search_repo_jobs`)
	return store.Exec(context.Background(), q)
}