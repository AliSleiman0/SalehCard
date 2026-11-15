// How often a page the customer is looking at re-checks data that can change
// on the server without them doing anything (an admin approves a top-up,
// completes or refunds an order, adjusts the balance). React Query pauses the
// timer while the tab is hidden (refetchIntervalInBackground defaults to false),
// and only the pages that opt in poll — the header's balance pill doesn't.
export const LIVE_REFETCH_MS = 30_000
