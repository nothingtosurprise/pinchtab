# Group 38: Pricing table

### 38.1 Read just the Pro plan
Navigate to `http://fixtures/pricing.html`. The page has three plan cards (`#plan-free`, `#plan-pro`, `#plan-enterprise`). Read only the Pro plan using a scoped snapshot.

**Verify**: Scoped snapshot contains `PLAN_PRO_PRICE_29` and `PLAN_PRO_LIMIT_5000_requests per day`.

### 38.2 Compare all three plans
Extract the full page (pricing grids are a Readability anti-pattern — reach for `--full`).

**Verify**: Output contains `PLAN_FREE_PRICE_0`, `PLAN_PRO_PRICE_29`, and `PLAN_ENTERPRISE_PRICE_CUSTOM`.
