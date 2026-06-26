// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Arabic (`ar`).
class AppLocalizationsAr extends AppLocalizations {
  AppLocalizationsAr([String locale = 'ar']) : super(locale);

  @override
  String get appTitle => 'صالح كارد';

  @override
  String get emailLabel => 'البريد الإلكتروني';

  @override
  String get passwordLabel => 'كلمة المرور';

  @override
  String get signInButton => 'تسجيل الدخول';

  @override
  String get catalogTitle => 'المنتجات';

  @override
  String get retry => 'إعادة المحاولة';

  @override
  String get logout => 'تسجيل الخروج';

  @override
  String get outOfStock => 'غير متوفر';

  @override
  String get emptyCatalog => 'لا توجد منتجات متاحة.';

  @override
  String get productDetailUnavailable => 'هذا المنتج غير متاح حالياً.';

  @override
  String get languageToggle => 'English';

  @override
  String get welcomeBackTitle => 'مرحباً بعودتك';

  @override
  String get signInSubtitle => 'سجّل الدخول برقم هاتفك.';

  @override
  String get getStartedTitle => 'لنبدأ';

  @override
  String get getStartedSubtitle =>
      'أدخل رقم هاتفك المحمول. سنرسل لك رمز التأكيد عليه.';

  @override
  String get createPasswordTitle => 'أنشئ كلمة المرور';

  @override
  String get createPasswordSubtitle => 'اختر كلمة مرور قوية لتأمين حسابك.';

  @override
  String get verifyTitle => 'تأكيد رقمك';

  @override
  String otpSubtitle(String phone) {
    return 'أدخل الرمز المكوّن من 6 أرقام المرسل إلى $phone.';
  }

  @override
  String get continueButton => 'متابعة';

  @override
  String get verifyButton => 'تأكيد';

  @override
  String get mobileNumberLabel => 'رقم الهاتف';

  @override
  String get phoneHint => '70 123 456';

  @override
  String get passwordHint => '••••••••';

  @override
  String get newPasswordHint => '٦ أحرف على الأقل';

  @override
  String get confirmPasswordLabel => 'تأكيد كلمة المرور';

  @override
  String get confirmPasswordHint => 'أعد إدخال كلمة المرور';

  @override
  String get termsPrefix => 'بإدخال رقم هاتفك، فإنك توافق على ';

  @override
  String get termsLink => 'الشروط والأحكام';

  @override
  String get resendPrefix => 'لم يصلك الرمز؟ ';

  @override
  String get resendLink => 'إعادة الإرسال';

  @override
  String get haveAccountPrefix => 'لديك حساب بالفعل؟ ';

  @override
  String get noAccountPrefix => 'ليس لديك حساب؟ ';

  @override
  String get signUpButton => 'إنشاء حساب';

  @override
  String get invalidPhone => 'يرجى إدخال رقم هاتف صحيح.';

  @override
  String get passwordRequired => 'يرجى إدخال كلمة المرور.';

  @override
  String get passwordTooShort =>
      'يجب أن تتكون كلمة المرور من 6 أحرف على الأقل.';

  @override
  String get passwordMismatch => 'كلمتا المرور غير متطابقتين.';

  @override
  String get otpIncomplete => 'أدخل الرمز المكوّن من 6 أرقام.';

  @override
  String get searchHint => 'ابحث';

  @override
  String get totalBalance => 'الرصيد الإجمالي';

  @override
  String get requestPhysicalCard => 'اطلب بطاقة فعلية';

  @override
  String get cardInfo => 'معلومات البطاقة';

  @override
  String get addMoney => 'إضافة رصيد';

  @override
  String get promoTitle => 'تسليم فوري';

  @override
  String get promoSubtitle =>
      'بطاقات الهدايا والتعبئة تصل إلى محفظتك خلال ثوانٍ.';

  @override
  String get featured => 'مميّز';

  @override
  String get navHome => 'الرئيسية';

  @override
  String get navCategories => 'الفئات';

  @override
  String get navCart => 'السلة';

  @override
  String get navMenu => 'القائمة';

  @override
  String get comingSoon => 'قريباً.';

  @override
  String get loadFailed => 'تعذّر تحميل المنتجات.';

  @override
  String get fromLabel => 'يبدأ من';

  @override
  String get chooseAmount => 'اختر الفئة';

  @override
  String get quantityLabel => 'الكمية';

  @override
  String get requiredBadge => 'مطلوب';

  @override
  String ratingsCount(int count) {
    return '($count تقييم)';
  }

  @override
  String enterValue(String field) {
    return 'أدخل $field';
  }

  @override
  String get deliveredInstantly => 'سنسلّم الرصيد إلى هذا الحساب فوراً.';

  @override
  String get addToCart => 'أضف للسلة';

  @override
  String get buyNow => 'اشترِ الآن';

  @override
  String get notifyMe => 'تنبيهي';

  @override
  String get totalLabel => 'الإجمالي';

  @override
  String get cartTitle => 'سلة المشتريات';

  @override
  String cartItemsCount(int count) {
    return '· $count عناصر';
  }

  @override
  String get cartEmptyTitle => 'سلتك فارغة';

  @override
  String get cartEmptySub => 'تصفّح الكتالوج وأضف بطاقات الهدايا والتعبئة.';

  @override
  String get browseCatalog => 'تصفّح الكتالوج';

  @override
  String get subtotalLabel => 'المجموع الفرعي';

  @override
  String get discountLabel => 'خصم';

  @override
  String get walletBalanceHint => 'رصيد المحفظة:';

  @override
  String get checkoutCta => 'إتمام الشراء';

  @override
  String get checkoutTitle => 'الدفع';

  @override
  String get paymentFailed => 'فشل الدفع. يرجى تجربة طريقة دفع أخرى.';

  @override
  String get deliveryDetails => 'تفاصيل التسليم';

  @override
  String get paymentMethodLabel => 'طريقة الدفع';

  @override
  String get payCardTitle => 'بطاقة';

  @override
  String get payCardSub => 'موافقة فورية';

  @override
  String get payWalletTitle => 'المحفظة';

  @override
  String get balanceLabel => 'الرصيد';

  @override
  String get insufficientBalance => 'رصيد غير كافٍ';

  @override
  String get payUsdtTitle => 'USDT';

  @override
  String get payUsdtSub => 'موافقة فورية';

  @override
  String get promoCodePlaceholder => 'رمز الخصم';

  @override
  String get applyLabel => 'تطبيق';

  @override
  String get placeOrderCta => 'تأكيد الطلب';

  @override
  String get fieldRequired => 'هذا الحقل مطلوب.';

  @override
  String get orderCompletedTitle => 'تم إتمام الطلب';

  @override
  String get orderProcessingTitle => 'نُكمل طلبك';

  @override
  String get orderCompletedSub => 'تم تسليم رموزك. استمتع!';

  @override
  String get orderProcessingSub => 'طلبات إضافة الرصيد قد تستغرق بضع دقائق.';

  @override
  String get orderLabel => 'رقم الطلب';

  @override
  String get deliveredCodesLabel => 'الرموز المُسلّمة';

  @override
  String get giftCardLabel => 'بطاقة هدية';

  @override
  String get copyLabel => 'نسخ';

  @override
  String get copiedLabel => 'تم النسخ';

  @override
  String get orderProcessingNote =>
      'سنُعلمك فور اكتمال الطلب. يمكنك متابعة حالته من صفحة الطلب.';

  @override
  String get viewOrderCta => 'عرض الطلب';

  @override
  String get backToHomeCta => 'العودة للرئيسية';

  @override
  String get orderItemsLabel => 'العناصر';

  @override
  String get statusPending => 'قيد الانتظار';

  @override
  String get statusProcessing => 'قيد المعالجة';

  @override
  String get statusCompleted => 'مكتمل';

  @override
  String get statusFailed => 'فشل';

  @override
  String get statusRefunded => 'مُسترَد';

  @override
  String get ordersTitle => 'طلباتي';

  @override
  String get filterAll => 'الكل';

  @override
  String get ordersEmptyTitle => 'لا توجد طلبات بعد';

  @override
  String get ordersEmptySub => 'ستظهر مشترياتك هنا بمجرد إتمام أول طلب.';

  @override
  String get orderRefundedNote =>
      'تم استرداد هذا الطلب. أُعيد المبلغ إلى محفظتك.';

  @override
  String get orderTimelineLabel => 'المسار';

  @override
  String get walletTitle => 'المحفظة';

  @override
  String get currentBalance => 'الرصيد الحالي';

  @override
  String get topUpCta => 'إضافة رصيد';

  @override
  String get sendMoney => 'إرسال أموال';

  @override
  String get txHistory => 'سجل العمليات';

  @override
  String get walletEmptyTitle => 'لا توجد عمليات بعد';

  @override
  String get walletEmptySub =>
      'أضف رصيداً إلى محفظتك للبدء — ستظهر عملياتك هنا.';

  @override
  String get txTopUp => 'إضافة رصيد';

  @override
  String get txPurchase => 'عملية شراء';

  @override
  String get txRefund => 'استرداد';

  @override
  String get txAdjustment => 'تسوية';

  @override
  String get topUpAmount => 'مبلغ الإضافة';

  @override
  String get topUpVia => 'الإضافة عبر';

  @override
  String topUpSuccess(String amount) {
    return 'تمت إضافة $amount+ إلى محفظتك';
  }

  @override
  String get sendMoneyComingSoon => 'إرسال الأموال قريباً.';

  @override
  String get recipientLabel => 'المستلِم (هاتف أو بريد إلكتروني)';

  @override
  String get amountLabel => 'المبلغ';

  @override
  String get noteLabel => 'ملاحظة (اختياري)';

  @override
  String get sendCta => 'إرسال';

  @override
  String get darkMode => 'الوضع الداكن';

  @override
  String get profileTitle => 'الملف الشخصي';

  @override
  String get accountInfo => 'معلومات الحساب';

  @override
  String get roleLabel => 'الدور';

  @override
  String get savedPlayerIds => 'معرّفات اللاعبين المحفوظة';

  @override
  String get savedPlayerIdsEmptyTitle => 'لا توجد معرّفات محفوظة';

  @override
  String get savedPlayerIdsEmptySub =>
      'احفظ معرّفات اللاعب أو الحساب لإتمام شراء أسرع.';

  @override
  String get addPlayerId => 'إضافة';

  @override
  String get playerIdHint => 'أدخل معرّف اللاعب';

  @override
  String get editProfile => 'تعديل';

  @override
  String get cancel => 'إلغاء';

  @override
  String get saveChanges => 'حفظ التغييرات';

  @override
  String get profileSaved => 'تم تحديث الملف الشخصي.';
}
