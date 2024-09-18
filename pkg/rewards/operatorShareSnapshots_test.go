package rewards

import (
	"fmt"
	"github.com/Layr-Labs/go-sidecar/internal/config"
	"github.com/Layr-Labs/go-sidecar/internal/logger"
	"github.com/Layr-Labs/go-sidecar/internal/sqlite"
	"github.com/Layr-Labs/go-sidecar/internal/sqlite/migrations"
	"github.com/Layr-Labs/go-sidecar/internal/tests"
	"github.com/Layr-Labs/go-sidecar/pkg/utils"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"testing"
)

func setupOperatorShareSnapshot() (
	*config.Config,
	*gorm.DB,
	*zap.Logger,
	error,
) {
	cfg := tests.GetConfig()
	l, _ := logger.NewLogger(&logger.LoggerConfig{Debug: cfg.Debug})

	db, err := tests.GetSqliteDatabaseConnection(l)
	if err != nil {
		panic(err)
	}
	sqliteMigrator := migrations.NewSqliteMigrator(db, l)
	if err := sqliteMigrator.MigrateAll(); err != nil {
		l.Sugar().Fatalw("Failed to migrate", "error", err)
	}

	return cfg, db, l, err
}

func teardownOperatorShareSnapshot(grm *gorm.DB) {
	queries := []string{
		`delete from operator_shares`,
		`delete from blocks`,
	}
	for _, query := range queries {
		if res := grm.Exec(query); res.Error != nil {
			fmt.Printf("Failed to run query: %v\n", res.Error)
		}
	}
}

func hydrateOperatorShareSnapshotBlocks(grm *gorm.DB, l *zap.Logger) (int, error) {
	contents, err := tests.GetOperatorSharesBlocksSqlFile()

	if err != nil {
		return 0, err
	}

	res := grm.Exec(contents)
	if res.Error != nil {
		l.Sugar().Errorw("Failed to execute sql", "error", zap.Error(res.Error), zap.String("query", contents))
		return 0, res.Error
	}
	return len(contents), err
}

func hydrateOperatorShares(grm *gorm.DB, l *zap.Logger) (int, error) {
	contents, err := tests.GetOperatorSharesSqlFile()

	if err != nil {
		return 0, err
	}

	_, err = sqlite.WrapTxAndCommit[interface{}](func(tx *gorm.DB) (interface{}, error) {
		for i, content := range contents {
			res := grm.Exec(content)
			if res.Error != nil {
				l.Sugar().Errorw("Failed to execute sql", "error", zap.Error(res.Error), zap.String("query", content), zap.Int("lineNumber", i))
				return nil, res.Error
			}
		}
		return nil, nil
	}, grm, nil)
	return len(contents), err
}

func Test_OperatorShareSnapshots(t *testing.T) {
	cfg, grm, l, err := setupOperatorShareSnapshot()

	if err != nil {
		t.Fatal(err)
	}

	snapshotDate := "2024-09-01"

	t.Run("Should hydrate dependency tables", func(t *testing.T) {
		if _, err := hydrateOperatorShareSnapshotBlocks(grm, l); err != nil {
			t.Error(err)
		}
		if _, err := hydrateOperatorShares(grm, l); err != nil {
			t.Error(err)
		}
	})
	t.Run("Should generate operator share snapshots", func(t *testing.T) {
		rewards := NewRewardsCalculator(l, nil, grm, cfg)

		snapshots, err := rewards.GenerateOperatorShareSnapshots(snapshotDate)
		assert.Nil(t, err)

		expectedResults, err := tests.GetOperatorSharesExpectedResults()
		assert.Nil(t, err)

		assert.Equal(t, len(expectedResults), len(snapshots))

		if len(expectedResults) != len(snapshots) {
			t.Errorf("Expected %d snapshots, got %d", len(expectedResults), len(snapshots))

			lacksExpectedResult := make([]*OperatorShareSnapshots, 0)
			// Go line-by-line in the snapshot results and find the corresponding line in the expected results.
			// If one doesnt exist, add it to the missing list.
			for _, snapshot := range snapshots {
				match := utils.Find(expectedResults, func(expected *tests.OperatorShareExpectedResult) bool {
					if expected.Operator == snapshot.Operator &&
						expected.Strategy == snapshot.Strategy &&
						expected.Snapshot == snapshot.Snapshot &&
						expected.Shares == snapshot.Shares {

						return true
					}

					return false
				})
				if match == nil {
					lacksExpectedResult = append(lacksExpectedResult, snapshot)
				}
			}
			assert.Equal(t, 0, len(lacksExpectedResult))

			if len(lacksExpectedResult) > 0 {
				for i, window := range lacksExpectedResult {
					fmt.Printf("%d - Snapshot: %+v\n", i, window)
				}
			}
		}
	})
	t.Cleanup(func() {
		teardownOperatorShareSnapshot(grm)
	})
}
