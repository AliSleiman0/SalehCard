package chatbot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// SeedEntry is one starter question with its follow-ups (one level deep, the
// same shape the help bot renders).
type SeedEntry struct {
	Question  Text
	Answer    Text
	Keywords  []string
	FollowUps []SeedEntry
}

// StarterContent is the help bot's starter Q&A, written from how the platform
// actually works today. Admins edit or remove any of it in admin → Help bot.
//
// It is served to EVERY client, including the iOS build, which must never
// mention crypto, Whish or money transfer (App Review). TestStarterContent
// enforces that, so keep answers generic ("the methods on the Top up screen").
var StarterContent = []SeedEntry{
	{
		Question: Text{En: "How do I buy a product?", Ar: "كيف أشتري منتجاً؟"},
		Answer: Text{
			En: "1. Verify your identity once (Profile → Verification).\n2. Top up your wallet.\n3. Open the product, choose a package, fill in the details it asks for (for example your player ID) and tap Buy.\nYou pay from your wallet balance.",
			Ar: "١. وثّق هويتك مرة واحدة (الملف الشخصي ← التحقق من الهوية).\n٢. اشحن محفظتك.\n٣. افتح المنتج، اختر الباقة، أدخل البيانات المطلوبة (مثل رقم اللاعب) واضغط شراء.\nيتم الدفع من رصيد محفظتك.",
		},
		Keywords: []string{"buy", "purchase", "order", "شراء", "اشتري", "طلب"},
		FollowUps: []SeedEntry{
			{
				Question: Text{En: "How do I pay?", Ar: "كيف أدفع؟"},
				Answer: Text{
					En: "All purchases are paid from your wallet balance. Top up your wallet first, then buy.",
					Ar: "جميع المشتريات تُدفع من رصيد محفظتك. اشحن محفظتك أولاً ثم اشترِ.",
				},
				Keywords: []string{"pay", "payment", "دفع"},
			},
			{
				Question: Text{En: "Why can't I tap Buy?", Ar: "لماذا لا أستطيع الضغط على شراء؟"},
				Answer: Text{
					En: "Check that:\n• your identity is verified and approved,\n• your wallet balance covers the price,\n• every required field is filled in.\nFor some games we check your player ID first; Buy unlocks once the ID is confirmed.",
					Ar: "تأكد من:\n• أن هويتك موثّقة وتمت الموافقة عليها،\n• أن رصيد محفظتك يكفي للسعر،\n• أن جميع الحقول المطلوبة معبأة.\nفي بعض الألعاب نتحقق من رقم اللاعب أولاً، ويُفعَّل زر الشراء بعد تأكيده.",
				},
				Keywords: []string{"buy button", "cannot buy", "لا أستطيع"},
			},
		},
	},
	{
		Question: Text{En: "How do I top up my wallet?", Ar: "كيف أشحن محفظتي؟"},
		Answer: Text{
			En: "Go to Wallet → Top up, pick one of the payment methods shown, enter the amount and the details the method asks for (for example a receipt photo), then send the request. Your balance is credited once the payment is confirmed.",
			Ar: "اذهب إلى المحفظة ← إضافة رصيد، اختر إحدى طرق الدفع الظاهرة، أدخل المبلغ والبيانات المطلوبة (مثل صورة الإيصال) ثم أرسل الطلب. يُضاف الرصيد إلى محفظتك بعد تأكيد الدفع.",
		},
		Keywords: []string{"top up", "topup", "wallet", "balance", "deposit", "شحن", "محفظة", "رصيد"},
		FollowUps: []SeedEntry{
			{
				Question: Text{En: "My top-up is still pending", Ar: "طلب الشحن ما زال قيد الانتظار"},
				Answer: Text{
					En: "Our team checks each payment before crediting it. You can follow the status in Wallet. If it has been pending for a long time, contact support with the amount and the method you used.",
					Ar: "يتحقق فريقنا من كل دفعة قبل إضافتها. يمكنك متابعة الحالة في المحفظة. إذا طال الانتظار، تواصل مع الدعم مع ذكر المبلغ والطريقة المستخدمة.",
				},
				Keywords: []string{"pending", "waiting", "انتظار", "معلق"},
			},
			{
				Question: Text{En: "My top-up was rejected", Ar: "تم رفض طلب الشحن"},
				Answer: Text{
					En: "A request is rejected when the payment could not be matched, for example a wrong amount or an unclear receipt. Your balance was not changed. Send a new request with the correct details, or contact support.",
					Ar: "يُرفض الطلب عندما لا يمكن مطابقة الدفعة، مثل مبلغ خاطئ أو إيصال غير واضح. لم يتغير رصيدك. أرسل طلباً جديداً ببيانات صحيحة أو تواصل مع الدعم.",
				},
				Keywords: []string{"rejected", "declined", "مرفوض", "رفض"},
			},
		},
	},
	{
		Question: Text{En: "Why do I need to verify my identity?", Ar: "لماذا يجب أن أوثّق هويتي؟"},
		Answer: Text{
			En: "Identity verification is required once before your first purchase, to protect your account and keep the service safe. Go to Profile → Verification (or tap the banner on the home screen), fill in your details and upload a photo of your ID or passport.",
			Ar: "توثيق الهوية مطلوب مرة واحدة قبل أول عملية شراء، لحماية حسابك وأمان الخدمة. اذهب إلى الملف الشخصي ← التحقق من الهوية (أو اضغط على الشريط في الصفحة الرئيسية)، أدخل بياناتك وارفع صورة الهوية أو جواز السفر.",
		},
		Keywords: []string{"kyc", "verify", "verification", "identity", "id", "توثيق", "هوية", "تحقق"},
		FollowUps: []SeedEntry{
			{
				Question: Text{En: "Which documents are accepted?", Ar: "ما المستندات المقبولة؟"},
				Answer: Text{
					En: "An ID card or driving licence (front and back), or a passport (photo page). Make sure the photo is clear and all four corners are visible.",
					Ar: "بطاقة الهوية أو رخصة القيادة (الوجه والخلف)، أو جواز السفر (صفحة الصورة). تأكد أن الصورة واضحة والزوايا الأربع ظاهرة.",
				},
				Keywords: []string{"documents", "passport", "license", "مستندات", "جواز", "رخصة"},
			},
			{
				Question: Text{En: "My verification was rejected", Ar: "تم رفض التوثيق"},
				Answer: Text{
					En: "Open Profile → Verification to see why, then submit again with clear photos and details that match your document.",
					Ar: "افتح الملف الشخصي ← التحقق من الهوية لمعرفة السبب، ثم أعد الإرسال بصور واضحة وبيانات مطابقة للمستند.",
				},
				Keywords: []string{"kyc rejected", "رفض التوثيق"},
			},
		},
	},
	{
		Question: Text{En: "Where is my order?", Ar: "أين طلبي؟"},
		Answer: Text{
			En: "Open Orders to see every order and its status. Codes are shown on the order as soon as it completes. Top-ups sent to a game or app account show as Processing until they are delivered.",
			Ar: "افتح «طلباتي» لرؤية جميع طلباتك وحالتها. تظهر الأكواد على الطلب فور اكتماله. أما الشحن إلى حساب لعبة أو تطبيق فيظهر بحالة «قيد التنفيذ» حتى يتم التسليم.",
		},
		Keywords: []string{"order", "status", "code", "where", "طلب", "طلبي", "كود", "حالة"},
		FollowUps: []SeedEntry{
			{
				Question: Text{En: "What does Processing mean?", Ar: "ماذا تعني «قيد التنفيذ»؟"},
				Answer: Text{
					En: "Your payment went through and the top-up is being delivered to the account you entered. It changes to Completed when done. If it stays Processing for a long time, contact support with your order number.",
					Ar: "تمت عملية الدفع ويجري تسليم الشحن إلى الحساب الذي أدخلته. تتغير الحالة إلى «مكتمل» عند الانتهاء. إذا بقيت طويلاً، تواصل مع الدعم مع رقم الطلب.",
				},
				Keywords: []string{"processing", "قيد التنفيذ"},
			},
			{
				Question: Text{En: "My order failed or was refunded", Ar: "طلبي فشل أو تم استرجاعه"},
				Answer: Text{
					En: "When an order cannot be delivered, the amount is returned to your wallet balance, so you can buy again right away. You can see the refund on the order and in your wallet history.",
					Ar: "عندما يتعذر تسليم الطلب، يُعاد المبلغ إلى رصيد محفظتك لتتمكن من الشراء مجدداً فوراً. يمكنك رؤية الاسترجاع على الطلب وفي سجل المحفظة.",
				},
				Keywords: []string{"failed", "refund", "refunded", "فشل", "استرجاع", "استرداد"},
			},
			{
				Question: Text{En: "I entered the wrong player ID", Ar: "أدخلت رقم لاعب خاطئ"},
				Answer: Text{
					En: "Contact support right away with your order number and the correct ID. If the order has not been delivered yet we will try to help, but a top-up already sent to another account usually cannot be reversed. Always double-check the ID before buying.",
					Ar: "تواصل مع الدعم فوراً مع رقم الطلب والرقم الصحيح. إذا لم يتم التسليم بعد سنحاول المساعدة، لكن الشحن المرسل إلى حساب آخر لا يمكن استرجاعه عادةً. تأكد دائماً من الرقم قبل الشراء.",
				},
				Keywords: []string{"wrong id", "player id", "mistake", "رقم خاطئ", "رقم اللاعب", "خطأ"},
			},
		},
	},
	{
		Question: Text{En: "How do I sign in?", Ar: "كيف أسجّل الدخول؟"},
		Answer: Text{
			En: "Sign in with your phone number: we send you a one-time code by SMS. If you set a password, you can sign in with your phone number and password instead.",
			Ar: "سجّل الدخول برقم هاتفك: نرسل لك رمزاً لمرة واحدة عبر رسالة نصية. إذا عيّنت كلمة مرور، يمكنك الدخول برقم هاتفك وكلمة المرور بدلاً من ذلك.",
		},
		Keywords: []string{"login", "sign in", "password", "code", "otp", "دخول", "كلمة المرور", "رمز"},
		FollowUps: []SeedEntry{
			{
				Question: Text{En: "I didn't receive the code", Ar: "لم يصلني الرمز"},
				Answer: Text{
					En: "Check that the phone number is correct, wait a minute, then tap Resend. If it still doesn't arrive, contact support.",
					Ar: "تأكد أن رقم الهاتف صحيح، انتظر دقيقة ثم اضغط إعادة الإرسال. إذا لم يصل بعد ذلك، تواصل مع الدعم.",
				},
				Keywords: []string{"no code", "sms", "resend", "لم يصل", "رسالة"},
			},
		},
	},
	{
		Question: Text{En: "How do I contact support?", Ar: "كيف أتواصل مع الدعم؟"},
		Answer: Text{
			En: "Use the contact buttons below this chat. Include your order number if your question is about an order.",
			Ar: "استخدم أزرار التواصل أسفل هذه المحادثة. أرفق رقم الطلب إذا كان سؤالك عن طلب.",
		},
		Keywords: []string{"support", "contact", "help", "whatsapp", "دعم", "تواصل", "مساعدة", "واتساب"},
	},
}

// SeedResult reports what Seed did (or, in a dry run, would do).
type SeedResult struct {
	Created []string // English question text of each entry created
	Skipped []string // already present (matched on the English question)
}

// Seed inserts StarterContent, skipping any question whose English text is
// already present at the same level, so a re-run (or a run after an admin
// started adding their own entries) never duplicates. Follow-ups attach to the
// existing parent when the parent was skipped. With apply=false nothing is
// written.
func Seed(ctx context.Context, repo Repository, content []SeedEntry, apply bool, now time.Time) (SeedResult, error) {
	var res SeedResult
	existing, err := repo.List(ctx, false)
	if err != nil {
		return res, err
	}
	if len(existing)+countSeed(content) > maxEntries {
		return res, fmt.Errorf("seeding would exceed the %d-entry cap", maxEntries)
	}
	key := func(parent string, q string) string { return parent + "\x00" + strings.ToLower(strings.TrimSpace(q)) }
	have := map[string]Entry{}
	for _, e := range existing {
		p := ""
		if e.ParentID != nil {
			p = e.ParentID.Hex()
		}
		have[key(p, e.Question.En)] = e
	}

	var insert func(items []SeedEntry, parent *Entry, label string) error
	insert = func(items []SeedEntry, parent *Entry, label string) error {
		parentHex := ""
		if parent != nil {
			parentHex = parent.ID.Hex()
		}
		for i, s := range items {
			name := label + s.Question.En
			e, found := have[key(parentHex, s.Question.En)]
			if found {
				res.Skipped = append(res.Skipped, name)
			} else {
				in := Input{Question: s.Question, Answer: s.Answer, Keywords: s.Keywords, SortOrder: (i + 1) * 10, Enabled: true}
				norm, _, msg, err := in.normalize()
				if err != nil {
					return fmt.Errorf("%q: %s", s.Question.En, msg)
				}
				e = Entry{
					ID:       bson.NewObjectID(),
					Question: norm.Question, Answer: norm.Answer, Keywords: norm.Keywords,
					SortOrder: norm.SortOrder, Enabled: true, CreatedBy: "seed",
					CreatedAt: now, UpdatedAt: now,
				}
				if parent != nil {
					pid := parent.ID
					e.ParentID = &pid
				}
				if apply {
					if err := repo.Create(ctx, &e); err != nil {
						return fmt.Errorf("create %q: %w", s.Question.En, err)
					}
				}
				res.Created = append(res.Created, name)
			}
			if len(s.FollowUps) > 0 {
				if parent != nil {
					return fmt.Errorf("%q: follow-ups are one level deep", s.Question.En)
				}
				if err := insert(s.FollowUps, &e, "  ↳ "); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := insert(content, nil, ""); err != nil {
		return res, err
	}
	return res, nil
}

func countSeed(items []SeedEntry) int {
	n := 0
	for _, s := range items {
		n += 1 + countSeed(s.FollowUps)
	}
	return n
}
