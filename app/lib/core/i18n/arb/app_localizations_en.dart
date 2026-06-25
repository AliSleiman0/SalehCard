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
  String get emailLabel => 'Email';

  @override
  String get passwordLabel => 'Password';

  @override
  String get signInButton => 'Sign in';

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

  @override
  String get welcomeBackTitle => 'Welcome back';

  @override
  String get signInSubtitle => 'Sign in with your mobile number.';

  @override
  String get getStartedTitle => 'Get Started';

  @override
  String get getStartedSubtitle =>
      'Enter your mobile number. We will send you a confirmation code there.';

  @override
  String get createPasswordTitle => 'Create a password';

  @override
  String get createPasswordSubtitle =>
      'Choose a strong password to secure your account.';

  @override
  String get verifyTitle => 'Verify your number';

  @override
  String otpSubtitle(String phone) {
    return 'Enter the 6-digit code we sent to $phone.';
  }

  @override
  String get continueButton => 'Continue';

  @override
  String get verifyButton => 'Verify';

  @override
  String get mobileNumberLabel => 'Mobile Number';

  @override
  String get phoneHint => '70 123 456';

  @override
  String get passwordHint => '••••••••';

  @override
  String get newPasswordHint => 'At least 6 characters';

  @override
  String get confirmPasswordLabel => 'Confirm password';

  @override
  String get confirmPasswordHint => 'Re-enter password';

  @override
  String get termsPrefix => 'By entering your phone number, you agree to our ';

  @override
  String get termsLink => 'terms and conditions';

  @override
  String get resendPrefix => 'Didn\'t get a code? ';

  @override
  String get resendLink => 'Resend';

  @override
  String get haveAccountPrefix => 'Already have an account? ';

  @override
  String get noAccountPrefix => 'Don\'t have an account? ';

  @override
  String get signUpButton => 'Sign up';

  @override
  String get invalidPhone => 'Please enter a valid mobile number.';

  @override
  String get passwordRequired => 'Please enter your password.';

  @override
  String get passwordTooShort => 'Password must be at least 6 characters.';

  @override
  String get passwordMismatch => 'Passwords do not match.';

  @override
  String get otpIncomplete => 'Enter the 6-digit code.';
}
