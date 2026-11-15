package chatbot

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// TagAndroidOnly marks a knowledge-base doc that must never reach the iOS
// build (App Review: no crypto, Whish or money-transfer content on iOS). The
// assistant drops such docs when the client says it is iOS.
const TagAndroidOnly = "android-only"

// KBSeedDoc is one starter knowledge-base article.
type KBSeedDoc struct {
	Slug  string
	Title Text
	Body  Text
	Tags  []string
}

// StarterKB is the AI assistant's starter knowledge base, written from how the
// app works today (menu labels match the app). Admins edit, disable or extend
// it in admin → Help bot → Knowledge base. Docs without the "android-only" tag
// reach every client, including the telecom-only iOS build, so they must not
// mention games, player IDs or gift cards (nor crypto / transfers); anything
// game- or code-specific lives in docs tagged "android-only". TestStarterKB
// enforces both.
var StarterKB = []KBSeedDoc{
	{
		Slug:  "how-to-buy",
		Title: Text{En: "How to buy a product", Ar: "كيف أشتري منتجاً"},
		Body: Text{
			En: "1. Verify your identity once (Menu → Verification). It is required before your first purchase.\n2. Top up your wallet (Wallet → Top up).\n3. Open the product, choose a package, fill in the details it asks for (for example the phone number or account to recharge) and tap Buy.\nAll purchases are paid from your wallet balance; there is no card payment at checkout.\nIf you cannot tap Buy, check that your identity is verified and approved, that your wallet balance covers the price, and that every required field is filled in. Always double-check the number or account you enter before buying.",
			Ar: "١. وثّق هويتك مرة واحدة (القائمة ← التحقق من الهوية). التوثيق مطلوب قبل أول عملية شراء.\n٢. اشحن محفظتك (المحفظة ← إضافة رصيد).\n٣. افتح المنتج، اختر الباقة، أدخل البيانات المطلوبة (مثل رقم الهاتف أو الحساب المراد شحنه) واضغط شراء.\nجميع المشتريات تُدفع من رصيد محفظتك، ولا يوجد دفع بالبطاقة عند الشراء.\nإذا لم تستطع الضغط على شراء، تأكد أن هويتك موثّقة وتمت الموافقة عليها، وأن رصيد محفظتك يكفي للسعر، وأن جميع الحقول المطلوبة معبأة. تأكد دائماً من الرقم أو الحساب الذي أدخلته قبل الشراء.",
		},
		Tags: []string{"buy", "checkout"},
	},
	{
		Slug:  "wallet-top-up",
		Title: Text{En: "Topping up your wallet", Ar: "شحن المحفظة"},
		Body: Text{
			En: "Go to Wallet → Top up, pick one of the payment methods shown, enter the amount and the details the method asks for (for example a receipt photo), then send the request.\nTop-ups are not instant: our team checks each payment, then credits your balance. You can follow the request in Wallet; a pending request shows as \"Top-up pending approval\".\nIf a request is rejected, the payment could not be matched (for example a wrong amount or an unclear receipt) and your balance was not changed. Send a new request with the correct details, or contact support with the amount and the method you used if it has been pending for a long time.\nThe available methods and any exchange rates are the ones shown on the Top up screen.",
			Ar: "اذهب إلى المحفظة ← إضافة رصيد، اختر إحدى طرق الدفع الظاهرة، أدخل المبلغ والبيانات المطلوبة (مثل صورة الإيصال) ثم أرسل الطلب.\nالشحن ليس فورياً: يتحقق فريقنا من كل دفعة ثم يضيف الرصيد إلى محفظتك. يمكنك متابعة الطلب في المحفظة، ويظهر الطلب المعلّق بعبارة «بانتظار الموافقة».\nإذا رُفض الطلب فهذا يعني أنه تعذّر مطابقة الدفعة (مثل مبلغ خاطئ أو إيصال غير واضح) ولم يتغير رصيدك. أرسل طلباً جديداً ببيانات صحيحة، أو تواصل مع الدعم مع ذكر المبلغ والطريقة المستخدمة إذا طال الانتظار.\nطرق الدفع المتاحة وأسعار الصرف هي الظاهرة في شاشة إضافة الرصيد.",
		},
		Tags: []string{"wallet", "top-up", "balance"},
	},
	{
		Slug:  "identity-verification",
		Title: Text{En: "Identity verification", Ar: "التحقق من الهوية"},
		Body: Text{
			En: "Identity verification is required once before your first purchase, to protect your account and keep the service safe. You can browse the app without it.\nGo to Menu → Verification (or tap the banner on the home screen), fill in your details and upload clear photos of your document: an ID card or driving licence (front and back), or a passport (photo page). All four corners must be visible.\nOur team reviews the submission; the status shows in Menu → Verification as Under review, You're verified, or Verification failed.\nIf it failed, open Menu → Verification to see why, then submit again with clear photos and details that match your document.",
			Ar: "توثيق الهوية مطلوب مرة واحدة قبل أول عملية شراء، لحماية حسابك وأمان الخدمة. يمكنك تصفح التطبيق بدونه.\nاذهب إلى القائمة ← التحقق من الهوية (أو اضغط على الشريط في الصفحة الرئيسية)، أدخل بياناتك وارفع صوراً واضحة للمستند: بطاقة الهوية أو رخصة القيادة (الوجه والخلف)، أو جواز السفر (صفحة الصورة). يجب أن تظهر الزوايا الأربع.\nيراجع فريقنا الطلب، وتظهر الحالة في القائمة ← التحقق من الهوية: قيد المراجعة، أو تم التوثيق، أو فشل التوثيق.\nإذا فشل التوثيق، افتح القائمة ← التحقق من الهوية لمعرفة السبب، ثم أعد الإرسال بصور واضحة وبيانات مطابقة للمستند.",
		},
		Tags: []string{"kyc", "verification"},
	},
	{
		Slug:  "order-status",
		Title: Text{En: "Orders and their status", Ar: "الطلبات وحالاتها"},
		Body: Text{
			En: "Open Orders (Menu → Orders) to see every order and its status. You can search by order number or product.\nIf a product comes with a code, the code is shown on the order as soon as it is Completed.\nRecharges sent to a phone number or account show as Processing while they are being delivered, then change to Completed. If an order stays Processing for a long time, contact support with your order number.",
			Ar: "افتح «طلباتي» (القائمة ← طلباتي) لرؤية جميع طلباتك وحالتها. يمكنك البحث برقم الطلب أو اسم المنتج.\nإذا كان المنتج يأتي مع كود، يظهر الكود على الطلب فور اكتماله.\nالشحن المرسل إلى رقم هاتف أو حساب يظهر بحالة «قيد التنفيذ» أثناء تسليمه، ثم يتغير إلى «مكتمل». إذا بقي الطلب قيد التنفيذ طويلاً، تواصل مع الدعم مع رقم الطلب.",
		},
		Tags: []string{"orders", "status", "codes"},
	},
	{
		Slug:  "failed-orders-refunds",
		Title: Text{En: "Failed orders, refunds and wrong details", Ar: "الطلبات الفاشلة والاسترجاع والبيانات الخاطئة"},
		Body: Text{
			En: "When an order cannot be delivered, the amount is returned to your wallet balance, so you can buy again right away. The refund shows on the order and in your wallet history. Refunds go back to the wallet, not to the original payment method.\nIf you entered a wrong number or account, contact support right away with your order number and the correct details. If the order has not been delivered yet the team will try to help, but a recharge already delivered to another number or account usually cannot be reversed.\nOnly the support team can decide on a refund for a delivered order.",
			Ar: "عندما يتعذر تسليم الطلب، يُعاد المبلغ إلى رصيد محفظتك لتتمكن من الشراء مجدداً فوراً. يظهر الاسترجاع على الطلب وفي سجل المحفظة. يُعاد المبلغ إلى المحفظة وليس إلى وسيلة الدفع الأصلية.\nإذا أدخلت رقماً أو حساباً خاطئاً، تواصل مع الدعم فوراً مع رقم الطلب والبيانات الصحيحة. إذا لم يتم التسليم بعد سيحاول الفريق المساعدة، لكن الشحن المسلَّم إلى رقم أو حساب آخر لا يمكن استرجاعه عادةً.\nفريق الدعم وحده يقرر بشأن استرجاع طلب تم تسليمه.",
		},
		Tags: []string{"refund", "failed", "wrong details"},
	},
	{
		Slug:  "complaints",
		Title: Text{En: "Submitting a complaint", Ar: "تقديم شكوى"},
		Body: Text{
			En: "If something went wrong with an order, a top-up or your account, go to Menu → Complaints → Submit a complaint. Choose a category (an order, a wallet top-up, my account, or something else), optionally pick the order, describe the problem and attach a photo if it helps.\nOur support team replies inside the complaint and you get a notification when they do. You can follow all your complaints in Menu → Complaints.",
			Ar: "إذا حدثت مشكلة في طلب أو في شحن المحفظة أو في حسابك، اذهب إلى القائمة ← الشكاوى ← تقديم شكوى. اختر الفئة (طلب، أو شحن المحفظة، أو حسابي، أو شيء آخر)، واختر الطلب إن أردت، واشرح المشكلة وأرفق صورة إذا كانت مفيدة.\nيرد فريق الدعم داخل الشكوى وتصلك إشعار عند الرد. يمكنك متابعة جميع شكاويك في القائمة ← الشكاوى.",
		},
		Tags: []string{"complaint", "support"},
	},
	{
		Slug:  "sign-in-and-password",
		Title: Text{En: "Signing in and your password", Ar: "تسجيل الدخول وكلمة المرور"},
		Body: Text{
			En: "Sign in with your phone number: we send you a one-time code by SMS. If you set a password, you can sign in with your phone number and password instead.\nIf the code doesn't arrive, check that the phone number is correct, wait a minute, then tap Resend. If it still doesn't arrive, contact support.\nTo change your password, go to Menu → Change password, enter your current password and the new one (at least 8 characters). Your other signed-in devices stay signed in.\nIf your account has no password yet: sign out, tap Sign up, enter the same phone number, choose a password, then enter the SMS code.\nWe will never ask you for your password or SMS code — never share them with anyone.",
			Ar: "سجّل الدخول برقم هاتفك: نرسل لك رمزاً لمرة واحدة عبر رسالة نصية. إذا عيّنت كلمة مرور، يمكنك الدخول برقم هاتفك وكلمة المرور بدلاً من ذلك.\nإذا لم يصل الرمز، تأكد أن رقم الهاتف صحيح، انتظر دقيقة ثم اضغط إعادة الإرسال. إذا لم يصل بعد ذلك، تواصل مع الدعم.\nلتغيير كلمة المرور، اذهب إلى القائمة ← تغيير كلمة المرور، أدخل كلمة المرور الحالية والجديدة (٨ أحرف على الأقل). تبقى أجهزتك الأخرى مسجّلة الدخول.\nإذا لم يكن لحسابك كلمة مرور بعد: سجّل الخروج، اضغط إنشاء حساب، أدخل رقم الهاتف نفسه، اختر كلمة مرور، ثم أدخل رمز الرسالة.\nلن نطلب منك أبداً كلمة المرور أو رمز الرسالة — لا تشاركهما مع أي أحد.",
		},
		Tags: []string{"login", "password", "otp"},
	},
	{
		Slug:  "delete-account",
		Title: Text{En: "Deleting your account", Ar: "حذف الحساب"},
		Body: Text{
			En: "Go to Menu → Delete account. Deleting is permanent: the account can't be recovered and you are signed out on all devices. Your ID photos are permanently deleted; past orders are kept anonymized for accounting.\nBefore deleting, spend your wallet balance and wait for orders, payments or top-up requests that are still in progress. You can delete anyway, but any remaining balance is lost and cannot be refunded.",
			Ar: "اذهب إلى القائمة ← حذف الحساب. الحذف نهائي: لا يمكن استعادة الحساب ويتم تسجيل خروجك من جميع الأجهزة. تُحذف صور الهوية نهائياً، وتبقى الطلبات السابقة مجهولة الهوية لأغراض المحاسبة.\nقبل الحذف، استخدم رصيد محفظتك وانتظر انتهاء الطلبات أو المدفوعات أو طلبات الشحن الجارية. يمكنك الحذف على أي حال، لكن الرصيد المتبقي يضيع ولا يمكن استرجاعه.",
		},
		Tags: []string{"account", "delete"},
	},
	{
		Slug:  "app-settings",
		Title: Text{En: "Language, dark mode and notifications", Ar: "اللغة والوضع الداكن والإشعارات"},
		Body: Text{
			En: "Switch the app between English and Arabic, or turn dark mode on or off, from Menu. Your profile details are in Menu → Profile.\nThe app sends notifications about your orders, top-up requests, verification and complaint replies; you can see them all on the Notifications screen.",
			Ar: "يمكنك تبديل لغة التطبيق بين العربية والإنجليزية، أو تشغيل الوضع الداكن وإيقافه، من القائمة. بيانات ملفك الشخصي في القائمة ← الملف الشخصي.\nيرسل التطبيق إشعارات عن طلباتك وطلبات الشحن والتوثيق والردود على الشكاوى، ويمكنك رؤيتها جميعاً في شاشة الإشعارات.",
		},
		Tags: []string{"settings", "language", "notifications"},
	},
	{
		// Android only: the iOS build sells telecom recharges, not games or
		// gift cards.
		Slug:  "game-top-ups-and-codes",
		Title: Text{En: "Game top-ups, player IDs and gift card codes", Ar: "شحن الألعاب ورقم اللاعب وأكواد البطاقات"},
		Body: Text{
			En: "Game top-ups are sent to the player ID you enter at checkout. For some games the app checks the player ID first and shows the account name; Buy unlocks once the ID is confirmed. Always double-check the player ID before buying.\nIf you entered the wrong player ID, contact support right away with your order number and the correct ID. If the order has not been delivered yet the team will try to help, but a top-up already delivered to another player usually cannot be reversed.\nGift cards and vouchers come as codes: the code is shown on the order (Menu → Orders) as soon as it is Completed. In Orders you can also search by player ID.",
			Ar: "يُرسل شحن الألعاب إلى رقم اللاعب الذي تدخله عند الشراء. في بعض الألعاب يتحقق التطبيق من رقم اللاعب أولاً ويعرض اسم الحساب، ويُفعَّل زر الشراء بعد تأكيده. تأكد دائماً من رقم اللاعب قبل الشراء.\nإذا أدخلت رقم لاعب خاطئاً، تواصل مع الدعم فوراً مع رقم الطلب والرقم الصحيح. إذا لم يتم التسليم بعد سيحاول الفريق المساعدة، لكن الشحن المسلَّم إلى لاعب آخر لا يمكن استرجاعه عادةً.\nبطاقات الهدايا والقسائم تأتي على شكل أكواد: يظهر الكود على الطلب (القائمة ← طلباتي) فور اكتماله. يمكنك أيضاً البحث في طلباتي برقم اللاعب.",
		},
		Tags: []string{TagAndroidOnly, "games", "player id", "codes"},
	},
	{
		Slug:  "contact-support",
		Title: Text{En: "Contacting support", Ar: "التواصل مع الدعم"},
		Body: Text{
			En: "Use the contact buttons at the bottom of the Help Center (for example WhatsApp), or submit a complaint from Menu → Complaints. Include your order number if your question is about an order. The assistant cannot see your account, orders or wallet, so questions about a specific order or payment are best sent to support.",
			Ar: "استخدم أزرار التواصل في أسفل مركز المساعدة (مثل واتساب)، أو قدّم شكوى من القائمة ← الشكاوى. أرفق رقم الطلب إذا كان سؤالك عن طلب. لا يستطيع المساعد رؤية حسابك أو طلباتك أو محفظتك، لذا من الأفضل إرسال الأسئلة عن طلب أو دفعة محددة إلى الدعم.",
		},
		Tags: []string{"support", "contact"},
	},
}

// KBSeedResult reports what SeedKB did (or, in a dry run, would do).
type KBSeedResult struct {
	Created []string // slugs created
	Skipped []string // slugs already present (left untouched)
}

// SeedKB inserts the starter docs whose slug is missing. An existing doc with
// the same slug — possibly edited by an admin — is never touched, so a re-run
// is safe. It refuses when the result would exceed the knowledge-base budget.
// With apply=false nothing is written.
func SeedKB(ctx context.Context, repo KBRepository, content []KBSeedDoc, apply bool, now time.Time) (KBSeedResult, error) {
	var res KBSeedResult
	existing, err := repo.List(ctx, false)
	if err != nil {
		return res, err
	}
	have := map[string]bool{}
	for _, d := range existing {
		if d.Slug != "" {
			have[d.Slug] = true
		}
	}
	var add []KBDoc
	for i, s := range content {
		if have[s.Slug] {
			res.Skipped = append(res.Skipped, s.Slug)
			continue
		}
		in, msg, err := KBInput{Slug: s.Slug, Title: s.Title, Body: s.Body, Tags: s.Tags, SortOrder: (i + 1) * 10, Enabled: true}.normalize()
		if err != nil {
			return res, fmt.Errorf("%s: %s", s.Slug, msg)
		}
		if in.Slug == "" {
			return res, fmt.Errorf("starter doc %d needs a slug", i)
		}
		add = append(add, KBDoc{
			ID: bson.NewObjectID(), Slug: in.Slug, Title: in.Title, Body: in.Body, Tags: in.Tags,
			SortOrder: in.SortOrder, Enabled: true, CreatedBy: "seed", UpdatedBy: "seed",
			CreatedAt: now, UpdatedAt: now,
		})
	}
	if len(existing)+len(add) > maxKBDocs {
		return res, fmt.Errorf("seeding would exceed the %d-article cap", maxKBDocs)
	}
	if used := kbUsedChars(append(existing, add...), nil); used > maxKBChars {
		return res, fmt.Errorf("seeding would use %d of the %d-character knowledge-base budget", used, maxKBChars)
	}
	for i := range add {
		if apply {
			inserted, err := repo.InsertIfSlugMissing(ctx, &add[i])
			if err != nil {
				return res, fmt.Errorf("insert %s: %w", add[i].Slug, err)
			}
			if !inserted {
				res.Skipped = append(res.Skipped, add[i].Slug)
				continue
			}
		}
		res.Created = append(res.Created, add[i].Slug)
	}
	return res, nil
}
