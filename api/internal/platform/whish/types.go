package whish

// Request/response DTOs mirror the Whish Web Service Technical Specification
// v1.4.2. Whish signals logical success/failure through the JSON `status`
// boolean regardless of HTTP 200; `code` is an operation-specific failure code.

type initiateRequest struct {
	Amount             float64 `json:"amount"`
	Currency           string  `json:"currency"`
	Invoice            string  `json:"invoice"`
	ExternalID         int64   `json:"externalId"`
	SuccessCallbackURL string  `json:"successCallbackUrl"`
	FailureCallbackURL string  `json:"failureCallbackUrl"`
	SuccessRedirectURL string  `json:"successRedirectUrl"`
	FailureRedirectURL string  `json:"failureRedirectUrl"`
}

type initiateResponse struct {
	Status bool   `json:"status"`
	Code   string `json:"code"`
	Data   struct {
		CollectURL string `json:"collectUrl"`
	} `json:"data"`
}

type statusRequest struct {
	Currency   string `json:"currency"`
	ExternalID int64  `json:"externalId"`
}

type statusResponse struct {
	Status bool   `json:"status"`
	Code   string `json:"code"`
	Data   struct {
		CollectStatus    string `json:"collectStatus"`    // "success" | "failed" | "pending"
		PayerPhoneNumber any    `json:"payerPhoneNumber"` // Long per docs, decoded leniently
	} `json:"data"`
}
