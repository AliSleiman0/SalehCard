package chatbot

import (
	"context"
	"time"
)

// StarterKnowledge is the AI assistant's starter knowledge, written from how
// the app works today (menu labels match the app). Admins rewrite it freely in
// admin → Help bot. General reaches every client, including the telecom-only
// iOS build, so it must not mention games, player IDs or gift cards (nor
// crypto / transfers); that lives in AndroidOnly. TestStarterKnowledge
// enforces it. Paragraphs are separated by a blank line.
var StarterKnowledge = KnowledgeInput{
	General: `How to buy: 1. Verify your identity once (Menu → Verification); it is required before your first purchase. 2. Top up your wallet (Wallet → Top up). 3. Open the product, choose a package, fill in the details it asks for (for example the phone number or account to recharge) and tap Buy. All purchases are paid from the wallet balance; there is no card payment at checkout. If Buy can't be tapped: check that your identity is verified and approved, that the wallet balance covers the price, and that every required field is filled in. Always double-check the number or account you enter before buying.

Topping up the wallet: go to Wallet → Top up, pick one of the payment methods shown, enter the amount and the details the method asks for (for example a receipt photo), then send the request. Top-ups are not instant: our team checks each payment, then credits the balance. A pending request shows in Wallet as "Top-up pending approval". A rejected request means the payment could not be matched (for example a wrong amount or an unclear receipt) and the balance did not change; send a new request with the correct details, or contact support with the amount and method if it has been pending for a long time. The available methods and any exchange rates are the ones shown on the Top up screen.

Identity verification: required once before the first purchase, to protect the account; you can browse without it. Go to Menu → Verification (or tap the banner on the home screen), fill in your details and upload clear photos of your document: an ID card or driving licence (front and back), or a passport (photo page), with all four corners visible. Our team reviews it; the status shows in Menu → Verification as Under review, You're verified, or Verification failed. If it failed, open Menu → Verification to see why, then submit again with clear photos and details that match the document.

Orders: Menu → Orders lists every order and its status; you can search by order number or product. If a product comes with a code, the code is shown on the order as soon as it is Completed. Recharges sent to a phone number or account show as Processing while they are being delivered, then Completed. If an order stays Processing for a long time, contact support with the order number.

Failed orders and refunds: when an order cannot be delivered, the amount goes back to the wallet balance (not to the original payment method), so you can buy again right away; the refund shows on the order and in the wallet history. If you entered a wrong number or account, contact support right away with the order number and the correct details; if the order has not been delivered yet the team will try to help, but a recharge already delivered to another number or account usually cannot be reversed. Only the support team decides on refunds for delivered orders.

Complaints: if something went wrong with an order, a top-up or your account, go to Menu → Complaints → Submit a complaint. Choose a category (an order, a wallet top-up, my account, or something else), optionally pick the order, describe the problem and attach a photo if it helps. Support replies inside the complaint and you get a notification; all complaints are listed in Menu → Complaints.

Signing in and passwords: sign in with your phone number and a one-time SMS code, or with your phone number and password if you set one. If the code doesn't arrive, check the number, wait a minute and tap Resend; if it still doesn't arrive, contact support. To change the password: Menu → Change password, enter the current and the new password (at least 8 characters); other devices stay signed in. If the account has no password yet: sign out, tap Sign up, enter the same phone number, choose a password, then enter the SMS code. We never ask for your password or SMS code — never share them.

Deleting the account: Menu → Delete account. It is permanent: the account can't be recovered and you are signed out on all devices; ID photos are deleted and past orders are kept anonymized for accounting. Spend the wallet balance and wait for orders, payments or top-up requests in progress first — any remaining balance is lost and cannot be refunded.

App settings: switch between English and Arabic or turn dark mode on or off from Menu; profile details are in Menu → Profile. Notifications about orders, top-up requests, verification and complaint replies are listed on the Notifications screen.

Contacting support: use the contact buttons at the bottom of the Help Center (for example WhatsApp) or submit a complaint from Menu → Complaints, and include the order number when asking about an order. The assistant can't see your account, orders or wallet, so questions about a specific order or payment are best sent to support.`,

	AndroidOnly: `Game top-ups and player IDs: game top-ups are sent to the player ID entered at checkout. For some games the app checks the player ID first and shows the account name; Buy unlocks once the ID is confirmed. Always double-check the player ID before buying. If you entered the wrong player ID, contact support right away with the order number and the correct ID; a top-up already delivered to another player usually cannot be reversed. In Orders you can also search by player ID.

Gift cards and vouchers: they come as codes, shown on the order (Menu → Orders) as soon as it is Completed.`,
}

// SeedKnowledge writes content as the knowledge unless knowledge already
// exists (admin edits are never overwritten). In a dry run (apply=false) it
// only reports whether it would write.
func SeedKnowledge(ctx context.Context, repo KnowledgeRepository, content KnowledgeInput, apply bool, now time.Time) (bool, error) {
	in, msg, err := content.normalize()
	if err != nil {
		return false, &seedError{msg}
	}
	if !apply {
		cur, err := repo.Get(ctx)
		return cur == nil, err
	}
	return repo.InsertIfMissing(ctx, &Knowledge{General: in.General, AndroidOnly: in.AndroidOnly, UpdatedBy: "seed", UpdatedAt: now})
}

type seedError struct{ msg string }

func (e *seedError) Error() string { return "starter knowledge: " + e.msg }
