package migrate

import (
	"embed"
)

//go:embed *.sql
var migrations embed.FS

// a successful case
// func TestShouldUpdateStats(t *testing.T) {

// 	poolmock, err := pgxmock.NewPool()
// 	require.NoError(t, err)
// 	poolmock.ExpectBegin()
// 	poolmock.ExpectExec("CREATE TABLE IF NOT EXISTS migration").WillReturnResult(pgxmock.NewResult("CREATE DATABASE", 1))
// 	poolmock.ExpectExec("LOCK TABLE migration;").WillReturnResult(pgxmock.NewResult("LOCK TABLE", 1))
// 	files, err := migrations.ReadDir(".")
// 	require.NoError(t, err)
// 	fmt.Println(files)
// 	require.Equal(t, 1, len(files))
// 	// for _, f := range files {
// 	poolmock.ExpectExec("SELECT id").WithArgs(pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult(pgx.ErrNoRows.Error(), 0))
// 	// script, err := migrations.ReadFile(f.Name())
// 	require.NoError(t, err)
// 	poolmock.
// 		poolmock.ExpectExec("CREATE TABLE").WillReturnResult(pgxmock.NewResult("CREATE DATABASE", 1))
// 	poolmock.ExpectExec("INSERT INTO migration *").WillReturnResult(pgxmock.NewResult("INSERT", 1))
// 	// }
// 	// poolmock.ExpectExec("INSERT INTO product_viewers").WithArgs(2, 3).WillReturnResult(pgxmock.NewResult("INSERT", 1))
// 	poolmock.ExpectCommit()
// 	// pool

// 	err = MigrateFromFS(context.TODO(), poolmock, &migrations, zap.L())
// 	assert.NoError(t, err) // now we execute our method
// 	// if err = recordStats(mock, 2, 3); err != nil {
// 	// 	t.Errorf("error was not expected while updating stats: %s", err)
// 	// }

// 	// we make sure that all expectations were met
// 	if err := poolmock.ExpectationsWereMet(); err != nil {
// 		t.Errorf("there were unfulfilled expectations: %s", err)
// 	}
// }
