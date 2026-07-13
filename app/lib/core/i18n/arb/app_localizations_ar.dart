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
  String get sendCodeButton => 'إرسال الرمز';

  @override
  String get useCodeInstead => 'تسجيل الدخول برمز بدلاً من ذلك';

  @override
  String get usePasswordInstead => 'تسجيل الدخول بكلمة المرور بدلاً من ذلك';

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
  String get offersEmpty => 'لا توجد عروض حالياً. تحقق لاحقاً.';

  @override
  String get productDetailUnavailable => 'هذا المنتج غير متاح حالياً.';

  @override
  String get verifyChecking => 'جارٍ التحقق من المعرّف…';

  @override
  String get verifyFound => 'الحساب';

  @override
  String get verifyNotFound =>
      'تعذّر العثور على هذا المعرّف. يرجى التحقق والمحاولة مرة أخرى.';

  @override
  String get verifyUnavailable => 'تعذّر التحقق الآن — يمكنك المتابعة.';

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
    return 'أدخل الرمز المكوّن من ٦ أرقام المرسل إلى $phone.';
  }

  @override
  String get continueButton => 'متابعة';

  @override
  String get verifyButton => 'تأكيد';

  @override
  String get nameLabel => 'الاسم الكامل';

  @override
  String get nameHint => 'اسمك';

  @override
  String get nameRequired => 'الرجاء إدخال اسمك.';

  @override
  String get mobileNumberLabel => 'رقم الهاتف';

  @override
  String get phoneHint => '70 123 456';

  @override
  String get passwordHint => '••••••••';

  @override
  String get newPasswordHint => '٨ أحرف على الأقل';

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
      'يجب أن تتكون كلمة المرور من ٨ أحرف على الأقل.';

  @override
  String get passwordMismatch => 'كلمتا المرور غير متطابقتين.';

  @override
  String get otpIncomplete => 'أدخل الرمز المكوّن من ٦ أرقام.';

  @override
  String get searchHint => 'ابحث';

  @override
  String get totalBalance => 'الرصيد الإجمالي';

  @override
  String get addMoney => 'إضافة رصيد';

  @override
  String get promoTitle => 'تسليم فوري';

  @override
  String get promoSubtitle => 'تُسلَّم أكواد بطاقات الهدايا فوراً بعد الدفع.';

  @override
  String get featured => 'مميّز';

  @override
  String get navHome => 'الرئيسية';

  @override
  String get navCategories => 'الفئات';

  @override
  String get navCart => 'السلة';

  @override
  String get navOffers => 'العروض';

  @override
  String get navMenu => 'القائمة';

  @override
  String get comingSoon => 'قريباً.';

  @override
  String get loadFailed => 'تعذّر تحميل المنتجات.';

  @override
  String get fromLabel => 'يبدأ من';

  @override
  String get chooseAmount => 'اختر المبلغ';

  @override
  String get quantityLabel => 'الكمية';

  @override
  String qtyTotalLine(int count, String unitPrice, String total) {
    return '$count × $unitPrice = $total';
  }

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
  String get creditAfterProcessing =>
      'يُضاف الرصيد إلى هذا الحساب بعد معالجة طلبك.';

  @override
  String get addToCart => 'أضف للسلة';

  @override
  String get buyNow => 'اشترِ الآن';

  @override
  String get notifyMe => 'أبلغني';

  @override
  String get totalLabel => 'الإجمالي';

  @override
  String get cartTitle => 'سلة المشتريات';

  @override
  String cartItemsCount(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count عنصر',
      many: '$count عنصراً',
      few: '$count عناصر',
      two: 'عنصران',
      one: 'عنصر واحد',
    );
    return '· $_temp0';
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
  String get payWalletTitle => 'المحفظة';

  @override
  String get balanceLabel => 'الرصيد';

  @override
  String get insufficientBalance => 'رصيد غير كافٍ';

  @override
  String get topUpWalletCta => 'اشحن المحفظة';

  @override
  String get kycRequiredTitle => 'التحقق من الهوية مطلوب';

  @override
  String get kycRequiredBody =>
      'تحقّق من هويتك لتتمكن من الشراء. تستغرق العملية دقيقة واحدة وتُراجع من قبل فريقنا.';

  @override
  String get kycRequiredCta => 'التحقق من الهوية';

  @override
  String get kycBannerBody =>
      'تحقّق من حسابك لتتمكن من الشراء — تستغرق العملية دقيقة واحدة. اضغط للبدء.';

  @override
  String get kycBannerPending =>
      'طلب التحقق قيد المراجعة. يمكنك الشراء فور الموافقة عليه.';

  @override
  String get promoCodePlaceholder => 'رمز الخصم';

  @override
  String get applyLabel => 'تطبيق';

  @override
  String get placeOrderCta => 'تأكيد الطلب';

  @override
  String get fieldRequired => 'هذا الحقل مطلوب.';

  @override
  String get invalidLebanesePhone =>
      'أدخل رقم هاتف محمول لبناني صالح (مثال: 71 123 456).';

  @override
  String get amountNotANumber => 'أدخل رقمًا صالحًا.';

  @override
  String amountOutOfRange(String min, String max) {
    return 'أدخل مبلغًا بين $min و$max.';
  }

  @override
  String amountAtLeast(String min) {
    return 'أدخل مبلغًا لا يقل عن $min.';
  }

  @override
  String amountAtMost(String max) {
    return 'أدخل مبلغًا لا يزيد عن $max.';
  }

  @override
  String amountRangeHint(String min, String max) {
    return 'بين $min و$max';
  }

  @override
  String amountMinHint(String min) {
    return 'الحد الأدنى $min';
  }

  @override
  String amountMaxHint(String max) {
    return 'الحد الأقصى $max';
  }

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
  String get topUpVia => 'كيف ستدفع؟';

  @override
  String get topUpNoteHint => 'مرجع الدفع / ملاحظة (اختياري)';

  @override
  String get topUpRequested =>
      'تم إرسال الطلب — سيُضاف الرصيد إلى محفظتك بعد تأكيد الدفع من الإدارة.';

  @override
  String get topUpRequestsTitle => 'طلبات الشحن الخاصة بي';

  @override
  String get statusApproved => 'مقبول';

  @override
  String get statusRejected => 'مرفوض';

  @override
  String walletPendingTitle(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count طلبات شحن قيد الموافقة',
      one: 'طلب شحن قيد الموافقة',
    );
    return '$_temp0';
  }

  @override
  String get walletPendingHint =>
      'يُحدَّث الرصيد بعد أن يؤكد المشرف عملية الدفع.';

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
  String get loyaltyPoints => 'نقاط الولاء';

  @override
  String get verificationTitle => 'التحقق';

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
  String get playerIdLabelHint => 'التسمية (مثال: PUBG الأساسي)';

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

  @override
  String get browseTitle => 'الفئات';

  @override
  String categoryItemCount(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count عنصر',
      many: '$count عنصراً',
      few: '$count عناصر',
      two: 'عنصران',
      one: 'عنصر واحد',
      zero: 'لا عناصر',
    );
    return '$_temp0';
  }

  @override
  String get searchPromptTitle => 'ابحث في الكتالوج';

  @override
  String get searchPromptSub =>
      'اكتب للعثور على بطاقات الهدايا والتعبئة والمزيد.';

  @override
  String get searchEmptyTitle => 'لا نتائج';

  @override
  String get searchEmptySub => 'جرّب كلمة مختلفة أو تصفّح الفئات.';

  @override
  String get notificationsTitle => 'الإشعارات';

  @override
  String get notificationsEmptyTitle => 'لا جديد لديك';

  @override
  String get notificationsEmptySub => 'ستظهر هنا إشعارات طلباتك ومحفظتك.';

  @override
  String get notifOrderCompletedTitle => 'تم إتمام الطلب';

  @override
  String notifOrderCompletedBody(String amount) {
    return 'تم تسليم طلبك — $amount.';
  }

  @override
  String get notifOrderRefundedTitle => 'تم استرداد الطلب';

  @override
  String notifOrderRefundedBody(String amount) {
    return 'تمت إعادة $amount إلى محفظتك.';
  }

  @override
  String get notifWalletTopUpTitle => 'تمت إضافة الرصيد';

  @override
  String notifWalletTopUpBody(String amount) {
    return 'تمت الموافقة على تعبئتك بقيمة $amount وإضافتها إلى محفظتك.';
  }

  @override
  String get notifTopUpRejectedTitle => 'تم رفض التعبئة';

  @override
  String notifTopUpRejectedBody(String amount) {
    return 'تم رفض طلب تعبئتك بقيمة $amount.';
  }

  @override
  String get notifKycApprovedTitle => 'تم التحقق من هويتك';

  @override
  String get notifKycApprovedBody => 'تم التحقق من حسابك — يمكنك الشراء الآن.';

  @override
  String get notifKycRejectedTitle => 'تم رفض التحقق';

  @override
  String get notifKycRejectedBody => 'يرجى إعادة إرسال معلوماتك.';

  @override
  String get notifPromoTitle => 'عرض لفترة محدودة';

  @override
  String get notifPromoBody => 'اكتشف أحدث بطاقات الهدايا والعروض.';

  @override
  String get kycTitle => 'التحقق';

  @override
  String get kycMenuLabel => 'التحقق من الهوية';

  @override
  String get kycBadgeUnverified => 'غير موثّق';

  @override
  String get kycBadgePending => 'قيد المراجعة';

  @override
  String get kycBadgeVerified => 'موثّق';

  @override
  String get kycBadgeRejected => 'مرفوض';

  @override
  String get kycUnverifiedTitle => 'وثّق هويتك';

  @override
  String get kycUnverifiedBody =>
      'وثّق هويتك للحصول على حدود أعلى وإتمام شراء أسرع. لن يستغرق الأمر سوى دقيقة.';

  @override
  String get kycPendingTitle => 'قيد المراجعة';

  @override
  String get kycPendingBody =>
      'لقد استلمنا مستنداتك وفريقنا يراجعها الآن. عادةً ما يستغرق هذا بضع دقائق.';

  @override
  String get kycVerifiedTitle => 'تم توثيق هويتك';

  @override
  String get kycVerifiedBody =>
      'تم تأكيد هويتك. كل الميزات مفعّلة — استمتع بحدود أعلى وإتمام شراء أسرع.';

  @override
  String get kycRejectedTitle => 'فشل التحقق';

  @override
  String get kycRejectedBody =>
      'تعذّر علينا التحقق من هويتك من المستندات المقدّمة. يرجى مراجعة البيانات وإعادة الإرسال.';

  @override
  String get kycVerifyNowCta => 'وثّق الآن';

  @override
  String get kycResubmitCta => 'إعادة الإرسال';

  @override
  String get kycFormTitle => 'التحقق من الهوية';

  @override
  String get kycFullNameLabel => 'الاسم الكامل';

  @override
  String get kycDobLabel => 'تاريخ الميلاد';

  @override
  String get kycDobHint => 'اختر تاريخ ميلادك';

  @override
  String get kycPlaceOfBirthLabel => 'مكان الولادة';

  @override
  String get kycPlaceOfResidenceLabel => 'مكان الإقامة';

  @override
  String get kycPlaceHint => 'المدينة، البلد';

  @override
  String get kycDocTypeLabel => 'نوع المستند';

  @override
  String get kycDocPassport => 'جواز سفر';

  @override
  String get kycDocIdCard => 'بطاقة هوية';

  @override
  String get kycDocLicense => 'رخصة قيادة';

  @override
  String get kycDocNumberLabel => 'رقم المستند';

  @override
  String get kycSubmitCta => 'إرسال';

  @override
  String get kycFieldRequired => 'هذا الحقل مطلوب.';

  @override
  String get kycSubmittedSnack => 'تم إرسال طلب التحقق — سنراجعه قريباً.';

  @override
  String get kycDocPhotosLabel => 'صور المستند';

  @override
  String get kycDocFrontLabel => 'وجه المستند';

  @override
  String get kycDocBackLabel => 'ظهر المستند';

  @override
  String get kycDocBackOptionalTag => 'اختياري لجوازات السفر';

  @override
  String get kycDocAddPhoto => 'إضافة صورة';

  @override
  String get kycDocReplacePhoto => 'استبدال الصورة';

  @override
  String get kycDocRemovePhoto => 'إزالة';

  @override
  String get kycDocUploading => 'جارٍ الرفع…';

  @override
  String get kycDocUploadFailed => 'فشل الرفع — اضغط لإعادة المحاولة.';

  @override
  String get kycDocPhotoRequired => 'هذه الصورة مطلوبة.';

  @override
  String get kycDocSourceCamera => 'التقاط صورة';

  @override
  String get kycDocSourceGallery => 'اختيار من المعرض';

  @override
  String get writeReview => 'اكتب مراجعة';

  @override
  String get reviewSheetTitle => 'قيّم هذا المنتج';

  @override
  String get reviewTapToRate => 'اضغط على نجمة للتقييم';

  @override
  String get reviewNoteOptionalHint => 'أخبرنا ما الذي حدث (اختياري)';

  @override
  String get reviewSubmit => 'إرسال المراجعة';

  @override
  String get reviewSubmittedPending => 'شكراً! مراجعتك قيد المراجعة والموافقة.';

  @override
  String get reviewAlreadyReviewed => 'لقد قمت بمراجعة هذا المنتج بالفعل.';

  @override
  String get reviewYouReviewed => 'لقد قمت بمراجعة هذا المنتج';

  @override
  String get commonDone => 'تم';

  @override
  String get usdtPayLabel => 'الدفع بعملة USDT (كريبتو)';

  @override
  String get usdtDepositTitle => 'إرسال USDT';

  @override
  String get usdtSendExactly =>
      'أرسل هذا المبلغ بالضبط — كل رقم مهم، وإلا لن تتم مطابقة دفعتك تلقائيًا:';

  @override
  String get usdtAmountLabel => 'المبلغ';

  @override
  String get usdtAmountCopied => 'تم نسخ المبلغ';

  @override
  String usdtAddressLabel(String network) {
    return 'عنوان الإيداع على شبكة $network';
  }

  @override
  String get usdtAddressCopied => 'تم نسخ العنوان';

  @override
  String usdtNetworkWarning(String network) {
    return 'أرسل عملة USDT فقط على شبكة $network. إرسال أي عملة أو شبكة أخرى سيؤدي إلى فقدان الأموال.';
  }

  @override
  String usdtExpiresIn(String time) {
    return 'تنتهي الصلاحية خلال $time';
  }

  @override
  String get usdtWaiting => 'بانتظار الدفع…';

  @override
  String get usdtConfirming => 'تم رصد الدفعة — جارٍ التأكيد على الشبكة…';

  @override
  String get usdtConfirmedTitle => 'تم تأكيد الدفع';

  @override
  String get usdtConfirmedBody => 'تم تأكيد دفعتك بعملة USDT.';

  @override
  String get usdtExpiredTitle => 'انتهت مهلة الدفع';

  @override
  String get usdtExpiredBody =>
      'لم يتم استلام أي دفعة في الوقت المحدد. ابدأ من جديد للحصول على عنوان جديد.';

  @override
  String get usdtNetworkLabel => 'الشبكة';

  @override
  String get privacyPolicyMenuLabel => 'سياسة الخصوصية';

  @override
  String get deleteAccountMenuLabel => 'حذف الحساب';

  @override
  String get deleteAccountTitle => 'حذف حسابك؟';

  @override
  String get deleteAccountWarnPermanent =>
      'هذا الإجراء نهائي — لا يمكن استعادة حسابك، وسيتم تسجيل خروجك من جميع الأجهزة.';

  @override
  String get deleteAccountWarnKycDeleted =>
      'تُحذف صور هويتك (مستندات التحقق) نهائياً.';

  @override
  String get deleteAccountWarnOrdersKept =>
      'تُحفظ طلباتك السابقة بصورة مجهولة الهوية وفق المتطلبات المحاسبية.';

  @override
  String get deleteAccountWarnWalletEmpty =>
      'يجب أن تكون محفظتك فارغة قبل حذف حسابك.';

  @override
  String deleteAccountConfirmPrompt(String word) {
    return 'للتأكيد، اكتب \"$word\" أدناه.';
  }

  @override
  String get deleteAccountConfirmWord => 'حذف';

  @override
  String get deleteAccountCta => 'حذف حسابي';

  @override
  String get deleteAccountWalletNotEmpty =>
      'لا يزال في محفظتك رصيد. استخدمه أولاً ثم حاول مرة أخرى.';

  @override
  String get deleteAccountOrdersInFlight =>
      'لديك طلبات قيد المعالجة. يرجى الانتظار حتى اكتمالها ثم المحاولة مرة أخرى.';

  @override
  String get deleteAccountPaymentsPending =>
      'لديك عملية دفع قيد التنفيذ. يرجى الانتظار حتى اكتمالها أو انتهاء مهلتها ثم المحاولة مرة أخرى.';

  @override
  String get deleteAccountTopUpsPending =>
      'لديك طلب شحن للمحفظة بانتظار الموافقة. يرجى انتظار البتّ فيه ثم المحاولة مرة أخرى.';
}
