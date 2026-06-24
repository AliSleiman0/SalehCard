// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for English (`en`).
class AppLocalizationsEn extends AppLocalizations {
  AppLocalizationsEn([String locale = 'en']) : super(locale);

  @override
  String get appTitle => 'SalehCard';

  @override
  String get loginTitle => 'Sign in';

  @override
  String get emailLabel => 'Email';

  @override
  String get passwordLabel => 'Password';

  @override
  String get signInButton => 'Sign in';

  @override
  String get loginFailed =>
      'Sign in failed. Check your credentials and try again.';

  @override
  String get catalogTitle => 'Products';

  @override
  String get retry => 'Retry';

  @override
  String get logout => 'Log out';

  @override
  String get outOfStock => 'Out of stock';

  @override
  String get emptyCatalog => 'No products available.';

  @override
  String get productDetailUnavailable =>
      'This product is currently unavailable.';

  @override
  String get languageToggle => 'العربية';
}
