# Design: axon-book as a Banking Ledger

**Date**: 2026-03-22
**Status**: Exploratory

## Premise

axon-book is already a general-purpose, event-sourced, double-entry ledger. The lemonade stand is a teaching example, not the product. The question: what separates it from a banking ledger?

**Answer**: Very little. One ledger enhancement; everything else is domain logic built on top.

## Ledger Enhancement: Effective Date

Add an optional `effective` date to `JournalEntryPosted`, defaulting to the posted date when not specified.

- **General ledger use**: accruals (expense incurred in March, recorded in April), backdated adjustments, period-end reporting ("entries effective in Q1" vs "entries posted in Q1")
- **Banking use**: value-dated entries for interest calculation (money received today, valued tomorrow)
- **Backwards compatible**: existing entries default effective = posted

The event timestamp from axon-fact provides a third date (processing/system time) for audit purposes.

## Banking Concerns That Are NOT Ledger Changes

| Concern | Where it lives | How it works |
|---|---|---|
| **Holds / reservations** | Banking reactor | Pattern of journal entries: `DR account:held / CR account`. Uses existing parent/child account structure. Release = reversal entry. |
| **Available balance** | Banking projection | `balance(account) - balance(account:held)`. A domain-specific query, not a ledger primitive. |
| **Idempotency** | Service/API layer | Deduplication on a client-provided key. Event store or API concern, not ledger logic. |
| **Interest accrual** | Banking reactor | Daily calculation based on value-dated balances. Posts accrual entries via the reactor pattern. |
| **Statement generation** | Banking projection | Read model built from journal entries, same pattern as DailySummaryProjection. |
| **Account types** (demand deposit, savings, loan, nostro/vostro) | Chart of accounts | Just accounts with appropriate types/naming. The ledger already supports arbitrary account types. |

## Architecture

The banking layer follows the same pattern as the lemonade stand example:

```
Banking Domain Events            axon-book Ledger
(hold.placed, payment.received) → Banking Reactor → JournalEntryPosted
                                                     ↓
                                              BalanceProjection
                                                     ↓
                                     Banking Projections (available balance, statements)
```

A banking application is a reactor + projections that use axon-book as its ledger engine.

## Parked Questions

- **Hierarchical balance aggregation**: The chart of accounts has parent/child structure. Should the ledger provide consolidated balance queries (sum account + children)? Useful for both GL (total assets) and banking (available balance). Deferred until there's a concrete use case.
- **Multi-tenancy**: If axon-book becomes a shared ledger service, namespace isolation vs separate databases. Not needed yet.
