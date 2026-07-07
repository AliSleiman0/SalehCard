package whish

import "context"

// Adapter implements [Provider] over the Whish HTTP [Client].
type Adapter struct {
	client *Client
}

// NewAdapter wraps a client as a Provider.
func NewAdapter(client *Client) *Adapter {
	return &Adapter{client: client}
}

// Initiate maps InitiateInput → the Whish initiate request and returns the
// collectUrl as both RedirectURL and ProviderRef (Whish has no separate ref).
func (a *Adapter) Initiate(ctx context.Context, in InitiateInput) (InitiateResult, error) {
	collectURL, err := a.client.Initiate(ctx, initiateRequest{
		Amount:             in.Amount,
		Currency:           in.Currency,
		Invoice:            in.Invoice,
		ExternalID:         in.ExternalID,
		SuccessCallbackURL: in.SuccessCallbackURL,
		FailureCallbackURL: in.FailureCallbackURL,
		SuccessRedirectURL: in.SuccessRedirectURL,
		FailureRedirectURL: in.FailureRedirectURL,
	})
	if err != nil {
		return InitiateResult{}, err
	}
	return InitiateResult{RedirectURL: collectURL, ProviderRef: collectURL}, nil
}

// GetStatus re-polls the Whish collect-status endpoint.
func (a *Adapter) GetStatus(ctx context.Context, q StatusQuery) (StatusResult, error) {
	status, phone, err := a.client.CollectStatus(ctx, statusRequest{
		Currency:   q.Currency,
		ExternalID: q.ExternalID,
	})
	if err != nil {
		return StatusResult{}, err
	}
	return StatusResult{Status: CollectStatus(status), PayerPhone: phone}, nil
}
