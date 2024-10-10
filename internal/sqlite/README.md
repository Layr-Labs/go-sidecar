

## Nile staker tokens expected values query

```sql
select
    staker_proportion,
    tokens_per_day,
    total_staker_operator_payout
from dbt_testnet_holesky_rewards."staker_rewards__2024-07-27 00:00:00_s_2024-07-28 13:00:00"
limit 100
```
