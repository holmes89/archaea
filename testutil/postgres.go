// Package testutil provides helpers for integration tests.
package testutil

import (
	"fmt"
	"log"
	"os"

	docker "github.com/ory/dockertest/v3/docker"

	dockertest "github.com/ory/dockertest/v3"
)

// SetupPostgres starts a postgres:16-alpine container and waits until the
// provided connect function returns nil. Returns the DSN (without the
// "postgres://" prefix) and a cleanup function that purges the container.
//
// Typical usage in TestMain:
//
//	func TestMain(m *testing.M) {
//	    dsn, cleanup := testutil.SetupPostgres("myuser", "mypass", "mydb",
//	        func(dsn string) error {
//	            db, err := repo.NewDatabase(dsn, false)
//	            if err != nil { return err }
//	            return db.Close()
//	        },
//	    )
//	    defer cleanup()
//	    // run migrations, then m.Run()
//	}
//
// Calls log.Fatalf on unrecoverable errors (container start failure, timeout).
func SetupPostgres(user, password, dbName string, connect func(dsn string) error) (dsn string, cleanup func()) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("testutil: could not connect to Docker: %v", err)
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "16-alpine",
		Env: []string{
			"POSTGRES_USER=" + user,
			"POSTGRES_PASSWORD=" + password,
			"POSTGRES_DB=" + dbName,
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("testutil: could not start postgres container: %v", err)
	}

	dsn = fmt.Sprintf("%s:%s@localhost:%s/%s?sslmode=disable",
		user, password, resource.GetPort("5432/tcp"), dbName)

	if err := pool.Retry(func() error { return connect(dsn) }); err != nil {
		pool.Purge(resource) //nolint:errcheck
		log.Fatalf("testutil: postgres never became ready: %v", err)
	}

	cleanup = func() {
		if err := pool.Purge(resource); err != nil {
			fmt.Fprintf(os.Stderr, "testutil: could not purge postgres container: %v\n", err)
		}
	}

	return dsn, cleanup
}
