package migrate

// import (
// 	"context"
// 	"embed"
// 	"testing"

// 	"github.com/pashagolub/pgxmock"
// 	"github.com/sirupsen/logrus"
// 	"github.com/stretchr/testify/require"
// )

// func TestMigrate(t *testing.T) {
// 	// t.Parallel()
// 	// ctrl := gomock.NewController(t)
// 	// defer ctrl.Finish()

// 	// // given
// 	// mockPool := pgxpoolmock.NewMockPgxPool(ctrl)
// 	// columns := []string{"id", "price"}
// 	// pgxRows := pgxpoolmock.NewRows(columns).AddRow(100, 100000.9).ToPgxRows()
// 	// // mockPool.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Return(pgxRows, nil)
// 	// orderDao := testdata.OrderDAO{
// 	// 	Pool: mockPool,
// 	// }

// 	mock, err := pgxmock.NewPool()
// 	if err != nil {
// 		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
// 	}
// 	defer mock.Close()
// 	mock.ExpectBegin()
// 	// mock.ExpectExec(`CREATE TABLE metrics (
// 	// 	target VARCHAR ( 50 ) UNIQUE NOT NULL,
// 	// 	data jsonb NOT NULL
// 	// 	);`)
// 	// mock.ExpectExec(`CREATE TABLE metrics (
// 	// 		target VARCHAR ( 50 ) UNIQUE NOT NULL,
// 	// 		data jsonb NOT NULL
// 	// 	);`)
// 	mock.ExpectCommit()
// 	logrus.Info(123)
// 	require.NoError(t, MigrateFromFS(context.Background(), mock, &migrations, logrus.StandardLogger()))
// }

// //go:embed *.sql
// var migrations embed.FS
