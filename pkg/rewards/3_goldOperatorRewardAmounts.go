package rewards

const _3_goldOperatorRewardAmountsQuery = `
insert into gold_3_operator_reward_amounts
WITH operator_token_sums AS (
  SELECT
    reward_hash,
    snapshot,
    token,
    tokens_per_day,
    avs,
    strategy,
    multiplier,
    reward_type,
    operator,
    sum_big(operator_tokens) OVER (PARTITION BY operator, reward_hash, snapshot) AS operator_tokens
  FROM gold_2_staker_reward_amounts
),
-- Dedupe the operator tokens across strategies for each operator, reward hash, and snapshot
distinct_operators AS (
  SELECT *
  FROM (
      SELECT *,
        -- We can use an arbitrary order here since the staker_weight is the same for each (operator, strategy, hash, snapshot)
        -- We use strategy ASC for better debuggability
        ROW_NUMBER() OVER (PARTITION BY reward_hash, snapshot, operator ORDER BY strategy ASC) as rn
      FROM operator_token_sums
  ) t
  WHERE rn = 1
)
SELECT * FROM distinct_operators
`

func (rc *RewardsCalculator) GenerateGoldOperatorRewardAmountsTable() error {
	res := rc.grm.Exec(_3_goldOperatorRewardAmountsQuery)
	if res.Error != nil {
		rc.logger.Sugar().Errorw("Failed to create gold_operator_reward_amounts", "error", res.Error)
		return res.Error
	}
	return nil
}

func (rc *RewardsCalculator) CreateGold3OperatorRewardsTable() error {
	query := `
		create table if not exists gold_3_operator_reward_amounts (
			reward_hash TEXT NOT NULL,
			snapshot DATE NOT NULL,
			token TEXT NOT NULL,
			tokens_per_day TEXT NOT NULL,
			avs TEXT NOT NULL,
			strategy TEXT NOT NULL,
			multiplier TEXT NOT NULL,
			reward_type TEXT NOT NULL,
			operator TEXT NOT NULL,
			operator_tokens TEXT NOT NULL,
			rn INTEGER NOT NULL
		)
	`
	res := rc.grm.Exec(query)
	if res.Error != nil {
		rc.logger.Sugar().Errorw("Failed to create gold_3_operator_reward_amounts table", "error", res.Error)
		return res.Error
	}
	return nil
}
