package chatbot

import "strings"

// fullCatalog reports whether the client sells everything (Android app and the
// website). Anything else — "ios" or an unknown value — gets the restricted,
// telecom-only App Store build: no Android-only knowledge and the output guard
// below.
func fullCatalog(platform string) bool {
	return platform == "android" || platform == "web"
}

// restrictedTerms are subjects the iOS build must never show (App Review:
// crypto, Whish, money transfer, games and gift cards are not offered there).
// The iOS prompt already lacks that knowledge, but a model can still echo a
// term the customer typed ("can I pay with USDT?" → "if USDT is shown…"), so
// this is enforced on the output in code, not left to the prompt. Lowercase;
// matched as substrings, so keep entries specific enough not to hit ordinary
// words (e.g. "money transfer", not "transfer" — line-credit transfers are
// telecom).
var restrictedTerms = []string{
	// crypto
	"crypto", "usdt", "usdc", "bitcoin", "tether", "binance", "trc20", "bep20",
	"كريبتو", "عملات رقمية", "العملات الرقمية", "عملة رقمية", "بيتكوين", "تيثر",
	// wallets / transfers
	"whish", "money transfer", "تحويل أموال", "تحويل الأموال", "ويش موني",
	// games
	"game", "player id", "لعبة", "ألعاب", "الألعاب", "العاب", "اللاعب",
	// gift cards
	"gift card", "giftcard", "بطاقة هدايا", "بطاقات هدايا", "بطاقة هدية", "بطاقات الهدايا",
}

// restrictedHit returns the first restricted term in text ("" when clean).
func restrictedHit(text string) string {
	t := strings.ToLower(text)
	for _, term := range restrictedTerms {
		if strings.Contains(t, term) {
			return term
		}
	}
	return ""
}
