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
  String get loginTitle => 'تسجيل الدخول';

  @override
  String get emailLabel => 'البريد الإلكتروني';

  @override
  String get passwordLabel => 'كلمة المرور';

  @override
  String get signInButton => 'تسجيل الدخول';

  @override
  String get loginFailed => 'فشل تسجيل الدخول. تحقق من بياناتك وحاول مرة أخرى.';

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
}
