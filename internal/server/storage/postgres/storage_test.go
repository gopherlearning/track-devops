package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gopherlearning/track-devops/internal/metrics"
	"github.com/jackc/pgx/v4"
	"github.com/pashagolub/pgxmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestStorage(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	// s := &Storage{db: mock, logger: logger}
	t.Run("Ping", func(t *testing.T) {
		mock, err := pgxmock.NewPool(pgxmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer mock.Close()
		mock.ExpectPing().WillReturnError(ErrBD)
		mock.ExpectPing().WillDelayFor(4 * time.Second)
		mock.ExpectPing()
		s := &Storage{db: mock, logger: logger}
		assert.ErrorIs(t, s.Ping(context.TODO()), ErrBD)
		assert.ErrorIs(t, s.Ping(context.TODO()), ErrContextClosed)
		assert.NoError(t, s.Ping(context.TODO()))
		// we make sure that all expectations were met
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
	t.Run("GetMetric", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		s := &Storage{db: mock, logger: logger}
		tests := []struct {
			target string
			mType  metrics.MetricType
			name   string
			err    error
			rows   *pgxmock.Rows
			delta  *int64
			value  *float64
		}{
			{
				name:   "successCounter",
				target: "127.0.0.1",
				mType:  metrics.CounterType,
				rows:   mock.NewRows([]string{"id", "title", "body"}).AddRow("", int64(11), nil),
				delta:  metrics.GetInt64Pointer(11),
			},
			{
				name:   "successGauge",
				target: "127.0.0.1",
				mType:  metrics.GaugeType,
				rows:   mock.NewRows([]string{"id", "title", "body"}).AddRow("", nil, 1.1),
				value:  metrics.GetFloat64Pointer(1.1),
			},
			{
				name:   "errScan",
				target: "127.0.0.1",
				mType:  metrics.CounterType,
				err:    pgx.ErrNoRows,
			},
			{
				name:   "errMetricType",
				target: "127.0.0.1",
				rows:   mock.NewRows([]string{"id", "title", "body"}).AddRow("", nil, 1.1),
				mType:  metrics.MetricType("bla"),
				err:    metrics.ErrNoSuchMetricType,
			},
		}
		for _, v := range tests {
			t.Run(v.name, func(t *testing.T) {
				if v.rows != nil {
					mock.ExpectQuery(`^select (.+) from metrics where (.+)$`).WithArgs(v.target, v.name, v.mType).WillReturnRows(v.rows)
				} else {
					mock.ExpectQuery(`^select (.+) from metrics where (.+)$`).WithArgs(v.target, v.name, v.mType).WillReturnError(v.err)
				}
				m, err := s.GetMetric(context.TODO(), v.target, v.mType, v.name)
				if err != nil {
					assert.ErrorIs(t, err, v.err)
					return
				}
				assert.Equal(t, m.Delta, v.delta)
				assert.Equal(t, m.Value, v.value)

			})
		}
		// we make sure that all expectations were met
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("UpdateMetric", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		s := &Storage{db: mock, logger: logger}
		tests := []struct {
			name   string
			err1   error
			err2   error
			err3   error
			err4   error
			rows   *pgxmock.Rows
			target string
			tx     bool
		}{
			{
				name:   "dbError",
				target: "127.0.0.1",
				err1:   pgx.ErrTxClosed,
			},
			{
				name:   "successNoOld",
				target: "127.0.0.1",
				err2:   pgx.ErrTxClosed,
				rows:   &pgxmock.Rows{},
			},
			{
				name:   "prepareError",
				target: "127.0.0.1",
				err3:   pgx.ErrTxClosed,
				rows:   &pgxmock.Rows{},
			},
			// {
			// 	name:   "",
			// 	target: "127.0.0.1",
			// 	err:    nil,
			// 	rows:   nil,
			// },
			// {
			// 	name:   "",
			// 	target: "127.0.0.1",
			// 	err:    nil,
			// 	rows:   nil,
			// },
			// {
			// 	name:   "",
			// 	target: "127.0.0.1",
			// 	err:    nil,
			// 	rows:   nil,
			// },
			// {
			// 	name:   "",
			// 	target: "127.0.0.1",
			// 	err:    nil,
			// 	rows:   nil,
			// },
			// {
			// 	name:   "",
			// 	target: "127.0.0.1",
			// 	err:    nil,
			// 	rows:   nil,
			// },
		}
		for _, v := range tests {
			t.Run(v.name, func(t *testing.T) {
				func() {
					if v.err1 != nil {
						mock.ExpectQuery(`^select (.+) from metrics where(.+)$`).WillReturnError(v.err1)
						return
					}
					mock.ExpectQuery(`^select (.+) from metrics where(.+)$`).WillReturnRows(v.rows)
					if v.err2 != nil {
						mock.ExpectBegin().WillReturnError(v.err2)
						return
					}
					mock.ExpectBegin()
					if v.err3 != nil {
						mock.ExpectPrepare("insert", "^INSERT INTO metrics(.+)$").WillReturnError(v.err3)
						mock.ExpectRollback()
						fmt.Println(123)
						return
					}
					mock.ExpectPrepare("insert", "^INSERT INTO metrics(.+)$")
					mock.ExpectExec("insert").WithArgs(metrics.GetInt64Pointer(11), nil, "pointerTest", v.target)
					mock.ExpectPrepare("update", "^UPDATE metrics SET(.+)$")
					mock.ExpectCommit()
				}()
				err = s.UpdateMetric(context.TODO(), v.target, metrics.Metrics{Delta: metrics.GetInt64Pointer(11), ID: "pointerTest", MType: metrics.CounterType})
				switch {
				case v.err1 != nil:
					assert.ErrorContains(t, err, v.err1.Error())
				case v.err2 != nil:
					assert.ErrorContains(t, err, v.err2.Error())
				case v.err3 != nil:
					assert.ErrorContains(t, err, v.err3.Error())
				default:
					assert.NoError(t, err)
				}

			})
		}
	})
	t.Run("Close", func(t *testing.T) {
		s := &Storage{logger: logger}
		assert.NoError(t, s.Close(context.TODO()))
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		s = &Storage{db: mock, logger: logger}
		assert.NoError(t, s.Close(context.TODO()))
	})
	t.Run("List And Metrics", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		s := &Storage{db: mock, logger: logger}
		tests := []struct {
			name   string
			err    error
			rows   *pgxmock.Rows
			target string
		}{
			{
				name: "success",
				rows: mock.NewRows([]string{"target", "id", "hash", "mtype", "mdelta", "mvalue"}).AddRow("127.0.0.1", "counterTest", "", metrics.CounterType, int64(11), nil).AddRow("127.0.0.2", "gaugeTest", "", metrics.GaugeType, nil, float64(1.1)),
			},
			{
				name: "error",
				err:  pgx.ErrNoRows,
			},
			{
				name:   "successTarget",
				rows:   mock.NewRows([]string{"target", "id", "hash", "mtype", "mdelta", "mvalue"}).AddRow("127.0.0.1", "counterTest", "", metrics.CounterType, int64(11), nil).AddRow("127.0.0.1", "gaugeTest", "", metrics.GaugeType, nil, float64(1.1)),
				target: "127.0.0.1",
			},
		}
		for _, v := range tests {
			t.Run(v.name, func(t *testing.T) {
				if v.rows != nil {
					if len(v.target) != 0 {
						mock.ExpectQuery(`^select (.+) from metrics where(.+)$`).WillReturnRows(v.rows)
					} else {
						mock.ExpectQuery(`^select (.+) from metrics$`).WillReturnRows(v.rows)
					}
				} else {
					mock.ExpectQuery(`^select (.+) from metrics$`).WillReturnError(v.err)
				}
				if len(v.target) != 0 {
					mm, err := s.Metrics(context.TODO(), v.target)
					if err != nil {
						assert.ErrorContains(t, err, "queryRow failed: no rows in result set")
						return
					}
					assert.Equal(t, mm["127.0.0.1"][0].ID, "counterTest")
					assert.Equal(t, mm["127.0.0.1"][1].ID, "gaugeTest")
				} else {
					mm, err := s.List(context.TODO())
					if err != nil {
						assert.ErrorContains(t, err, "queryRow failed: no rows in result set")
						return
					}
					assert.Equal(t, mm["127.0.0.1"][0], "counter - counterTest - 11")
					assert.Equal(t, mm["127.0.0.2"][0], "gauge - gaugeTest - 1.1")
				}

			})
		}
		// we make sure that all expectations were met
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
