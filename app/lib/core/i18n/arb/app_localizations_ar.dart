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
}
