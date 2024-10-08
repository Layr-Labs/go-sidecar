package rewards

import (
	"fmt"
	"github.com/Layr-Labs/go-sidecar/internal/config"
	"github.com/Layr-Labs/go-sidecar/internal/logger"
	"github.com/Layr-Labs/go-sidecar/internal/sqlite/migrations"
	"github.com/Layr-Labs/go-sidecar/internal/tests"
	"github.com/Layr-Labs/go-sidecar/internal/tests/sqlite"
	"github.com/Layr-Labs/go-sidecar/internal/types/numbers"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// const TOTAL_BLOCK_COUNT = 1229187

func rewardsTestsEnabled() bool {
	return os.Getenv("TEST_REWARDS") == "true"
}

func getRewardsTestContext() string {
	ctx := os.Getenv("REWARDS_TEST_CONTEXT")
	if ctx == "" {
		return "testnet"
	}
	return ctx
}

func getProjectRootPath() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	p, err := filepath.Abs(fmt.Sprintf("%s/../..", wd))
	if err != nil {
		panic(err)
	}
	return p
}

func getSnapshotDate() (string, error) {
	context := getRewardsTestContext()

	switch context {
	case "testnet":
		return "2024-09-01", nil
	case "testnet-reduced":
		return "2024-07-25", nil
	case "mainnet-reduced":
		return "2024-08-01", nil
	}
	return "", fmt.Errorf("Unknown context: %s", context)
}

func hydrateAllBlocksTable(grm *gorm.DB, l *zap.Logger) (int, error) {
	projectRoot := getProjectRootPath()
	contents, err := tests.GetAllBlocksSqlFile(projectRoot)

	if err != nil {
		return 0, err
	}

	count := len(strings.Split(strings.Trim(contents, "\n"), "\n")) - 1

	res := grm.Exec(contents)
	if res.Error != nil {
		l.Sugar().Errorw("Failed to execute sql", "error", zap.Error(res.Error))
		return count, res.Error
	}
	return count, nil
}

func getRowCountForTable(grm *gorm.DB, tableName string) (int, error) {
	query := fmt.Sprintf("select count(*) as cnt from %s", tableName)
	var count int
	res := grm.Raw(query).Scan(&count)

	if res.Error != nil {
		return 0, res.Error
	}
	return count, nil
}

func setupRewards() (
	string,
	*config.Config,
	*gorm.DB,
	*zap.Logger,
	error,
) {
	cfg := tests.GetConfig()
	cfg.Chain = config.Chain_Holesky
	cfg.Debug = true
	l, _ := logger.NewLogger(&logger.LoggerConfig{Debug: cfg.Debug})

	dbFileName, db, err := sqlite.GetFileBasedSqliteDatabaseConnection(l)
	if err != nil {
		panic(err)
	}
	sqliteMigrator := migrations.NewSqliteMigrator(db, l)
	if err := sqliteMigrator.MigrateAll(); err != nil {
		l.Sugar().Fatalw("Failed to migrate", "error", err)
	}

	return dbFileName, cfg, db, l, err
}

func teardownRewards(grm *gorm.DB) {
	teardownOperatorAvsRegistrationSnapshot(grm)
	teardownOperatorAvsStrategyWindows(grm)
	teardownOperatorShareSnapshot(grm)
	teardownStakerDelegationSnapshot(grm)
	teardownStakerShareSnapshot(grm)
}

func Test_Rewards(t *testing.T) {
	if !rewardsTestsEnabled() {
		t.Skipf("Skipping %s", t.Name())
		return
	}

	if err := numbers.InitPython(); err != nil {
		t.Fatal(err)
	}

	dbFileName, cfg, grm, l, err := setupRewards()
	fmt.Printf("Using db file: %+v\n", dbFileName)

	if err != nil {
		t.Fatal(err)
	}

	snapshotDate, err := getSnapshotDate()
	if err != nil {
		t.Fatal(err)
	}

	t.Run("Should initialize the rewards calculator", func(t *testing.T) {
		rc, err := NewRewardsCalculator(l, grm, cfg)
		assert.Nil(t, err)
		if err != nil {
			t.Fatal(err)
		}
		assert.NotNil(t, rc)

		fmt.Printf("DB Path: %+v", dbFileName)

		query := `select name from main.sqlite_master where type = 'table' order by name asc`
		type row struct{ Name string }
		var tables []row
		res := rc.grm.Raw(query).Scan(&tables)
		assert.Nil(t, res.Error)

		expectedTables := []string{
			"combined_rewards",
			"gold_1_active_rewards",
			"gold_2_staker_reward_amounts",
			"gold_3_operator_reward_amounts",
			"gold_4_rewards_for_all",
			"gold_5_rfae_stakers",
			"gold_6_rfae_operators",
			"gold_7_staging",
			"gold_table",
			"operator_avs_registration_snapshots",
			"operator_avs_strategy_snapshots",
			"operator_share_snapshots",
			"staker_delegation_snapshots",
			"staker_share_snapshots",
		}
		tablesList := make([]string, 0)
		for i, table := range tables {
			fmt.Printf("[%v]: %+v\n", i, table.Name)
			tablesList = append(tablesList, table.Name)
		}

		for _, table := range expectedTables {
			assert.True(t, slices.Contains(tablesList, table))
		}

		// Setup all tables and source data
		_, err = hydrateAllBlocksTable(grm, l)
		assert.Nil(t, err)

		err = hydrateOperatorAvsStateChangesTable(grm, l)
		assert.Nil(t, err)

		err = hydrateOperatorAvsRestakedStrategies(grm, l)
		assert.Nil(t, err)

		err = hydrateOperatorShares(grm, l)
		assert.Nil(t, err)

		err = hydrateStakerDelegations(grm, l)
		assert.Nil(t, err)

		err = hydrateStakerShares(grm, l)
		assert.Nil(t, err)

		err = hydrateRewardSubmissionsTable(grm, l)
		assert.Nil(t, err)

		t.Log("Hydrated tables")

		// Generate snapshots
		err = rc.generateSnapshotData(snapshotDate)
		assert.Nil(t, err)

		t.Log("Generated and inserted snapshots")
		forks, err := cfg.GetForkDates()
		assert.Nil(t, err)

		startDate := "1970-01-01"
		err = rc.Generate1ActiveRewards(snapshotDate, startDate)
		assert.Nil(t, err)
		rows, err := getRowCountForTable(grm, "gold_1_active_rewards")
		assert.Nil(t, err)
		fmt.Printf("Rows in gold_1_active_rewards: %v\n", rows)

		err = rc.GenerateGold2StakerRewardAmountsTable(forks)
		assert.Nil(t, err)
		rows, err = getRowCountForTable(grm, "gold_2_staker_reward_amounts")
		assert.Nil(t, err)
		fmt.Printf("Rows in gold_2_staker_reward_amounts: %v\n", rows)

		err = rc.GenerateGold3OperatorRewardAmountsTable()
		assert.Nil(t, err)
		rows, err = getRowCountForTable(grm, "gold_3_operator_reward_amounts")
		assert.Nil(t, err)
		fmt.Printf("Rows in gold_3_operator_reward_amounts: %v\n", rows)

		err = rc.GenerateGold4RewardsForAllTable()
		assert.Nil(t, err)
		rows, err = getRowCountForTable(grm, "gold_4_rewards_for_all")
		assert.Nil(t, err)
		fmt.Printf("Rows in gold_4_rewards_for_all: %v\n", rows)

		err = rc.GenerateGold5RfaeStakersTable()
		assert.Nil(t, err)
		rows, err = getRowCountForTable(grm, "gold_5_rfae_stakers")
		assert.Nil(t, err)
		fmt.Printf("Rows in gold_5_rfae_stakers: %v\n", rows)

		err = rc.GenerateGold6RfaeOperatorsTable()
		assert.Nil(t, err)
		rows, err = getRowCountForTable(grm, "gold_6_rfae_operators")
		assert.Nil(t, err)
		fmt.Printf("Rows in gold_6_rfae_operators: %v\n", rows)

		err = rc.GenerateGold7StagingTable()
		assert.Nil(t, err)
		rows, err = getRowCountForTable(grm, "gold_7_staging")
		assert.Nil(t, err)
		fmt.Printf("Rows in gold_7_staging: %v\n", rows)

		err = rc.GenerateGold8FinalTable()
		assert.Nil(t, err)
		rows, err = getRowCountForTable(grm, "gold_table")
		assert.Nil(t, err)
		fmt.Printf("Rows in gold_table: %v\n", rows)

		fmt.Printf("Done!\n\n")
		t.Cleanup(func() {
			teardownRewards(grm)
			// tests.DeleteTestSqliteDB(dbFileName)
		})
	})
}
