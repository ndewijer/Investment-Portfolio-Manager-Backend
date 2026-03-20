# IBKR Flex Trade Notes/Codes Reference

The `notes` field on a `<Trade>` element contains zero or more semicolon-separated codes.
Example: `notes="IA;P"` = two codes: `IA` and `P`.

Codes are multi-character tokens — `RI` is one code, not `R` + `I`.
When checking for a specific code, split on `;` and match the whole token.

Sources:
- [IBKR Guides — Codes (authoritative)](https://www.ibkrguides.com/reportingreference/reportguide/codes_default.htm)
- [ibflex enums.py (community, includes undocumented codes)](https://github.com/csingley/ibflex/blob/master/ibflex/enums.py)

---

## Codes seen in real data

| Code | Meaning | Source |
|------|---------|--------|
| `RI` | Recurring Investment (IBKR auto-invest feature) | IBKR Guides |
| `IA` | Executed against an IB affiliate | IBKR Guides |
| `FP` | IB acted as principal for the fractional share portion | ibflex |
| `P`  | Partial execution (order filled in multiple legs) | IBKR Guides |
| `""` | Regular manual order — no special classification | — |

> **Note:** `Ri` (lowercase i) and `RI` (uppercase I) are distinct codes — capitalisation matters.
> ibflex conflates them by mapping `RI` to "Reimbursement", but IBKR Guides correctly separates them:
> `Ri` = Reimbursement, `RI` = Recurring Investment. Real-world data confirms the distinction.

---

## Other codes likely to be encountered

| Code | Meaning | Notes |
|------|---------|-------|
| `R`  | Dividend Reinvestment (DRIP) | Use to identify DRIP buys when dividend matching is implemented |
| `O`  | Opening Trade | Regular buy opening a new position |
| `C`  | Closing Trade | Regular sell closing a position |
| `Co` | Corrected Trade | Amended version of a prior trade; check for re-imports |
| `Ca` | Cancelled | Shouldn't appear if Flex Query has "Include Cancelled Trades = NO" |
| `B`  | Automatic Buy-in | IB-forced purchase to close a short position |
| `M`  | Entered manually by IB | Manual correction by IB operations |
| `RP` | Riskless principal for fractional share portion | Similar to `FP`; ibflex only |
| `D`  | IB acted as Dual Agent | ibflex only, not in IBKR Guides |
| `AFx`| AutoFX conversion from trading | ibflex only, not in IBKR Guides |

---

## Tax lot codes (informational only, no action needed)

These appear on trades as accounting metadata. No special handling required.

| Code | Meaning |
|------|---------|
| `HC` | Highest Cost lot method |
| `LI` | Last In, First Out (LIFO) |
| `SL` | Specific Lot |
| `MLG` | Maximize Long-Term Gain |
| `MLL` | Maximize Long-Term Loss |
| `MSG` | Maximize Short-Term Gain |
| `MSL` | Maximize Short-Term Loss |
| `ML` | Maximize Losses |
| `LT` | Long-term P/L |
| `ST` | Short-term P/L |
| `LD` | Loss disallowed from Wash Sale |

---

## Full code reference (IBKR Guides)

All codes as documented at the time of writing. ibflex may include additional undocumented codes.

| Code | Meaning |
|------|---------|
| `A` | Assignment |
| `AEx` | Automatic exercise for dividend-related recommendation |
| `Adj` | Adjustment |
| `Al` | Allocation |
| `Aw` | Away Trade |
| `B` | Automatic Buy-in |
| `Bo` | Direct Borrow |
| `C` | Closing Trade |
| `CD` | Cash Delivery |
| `CP` | Complex Position |
| `Ca` | Cancelled |
| `Co` | Corrected Trade |
| `Cx` | Part or all executed as a Crossing by IB as dual agent |
| `ETF` | ETF Creation/Redemption |
| `Ep` | Resulted from an Expired Position |
| `Ex` | Exercise |
| `G` | Trade in Guaranteed Account Segment |
| `GEA` | Expiration or Assignment resulting from offsetting positions |
| `HC` | Highest Cost tax lot method |
| `HFI` | Investment Transferred to Hedge Fund |
| `HFR` | Redemption from Hedge Fund |
| `I` | Internal Transfer |
| `IA` | Executed against an IB affiliate |
| `INV` | Investment Transfer from Investor |
| `L` | Ordered by IB (Margin Violation) |
| `LD` | Adjusted by Loss Disallowed from Wash Sale |
| `LI` | Last In, First Out (LIFO) tax lot method |
| `LT` | Long-term P/L |
| `Lo` | Direct Loan |
| `M` | Entered manually by IB |
| `MEx` | Manual exercise for dividend-related recommendation |
| `ML` | Maximize Losses tax basis election |
| `MLG` | Maximize Long-Term Gain tax lot method |
| `MLL` | Maximize Long-Term Loss tax lot method |
| `MSG` | Maximize Short-Term Gain tax lot method |
| `MSL` | Maximize Short-Term Loss tax lot method |
| `O` | Opening Trade |
| `P` | Partial Execution |
| `PI` | Price Improvement |
| `Po` | Interest or Dividend Accrual Posting |
| `Pr` | Part or all executed as a Crossing by the Exchange as principal |
| `R` | Dividend Reinvestment |
| `Rb` | Rebill |
| `RED` | Redemption to Investor |
| `Re` | Interest or Dividend Accrual Reversal |
| `Ri` | Reimbursement |
| `RI` | Recurring Investment |
| `SI` | Order solicited by Interactive Brokers |
| `SL` | Specific Lot tax lot method |
| `SO` | Order marked as solicited by Introducing Broker |
| `SS` | Shortened settlement trade designation |
| `ST` | Short-term P/L |
| `SY` | Positions eligible for Stock Yield Enhancement Program |
| `T` | Transfer |
