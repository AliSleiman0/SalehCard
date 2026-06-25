import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:intl/intl.dart' as intl;

import 'app_localizations_ar.dart';
import 'app_localizations_en.dart';

// ignore_for_file: type=lint

/// Callers can lookup localized strings with an instance of AppLocalizations
/// returned by `AppLocalizations.of(context)`.
///
/// Applications need to include `AppLocalizations.delegate()` in their app's
/// `localizationDelegates` list, and the locales they support in the app's
/// `supportedLocales` list. For example:
///
/// ```dart
/// import 'arb/app_localizations.dart';
///
/// return MaterialApp(
///   localizationsDelegates: AppLocalizations.localizationsDelegates,
///   supportedLocales: AppLocalizations.supportedLocales,
///   home: MyApplicationHome(),
/// );
/// ```
///
/// ## Update pubspec.yaml
///
/// Please make sure to update your pubspec.yaml to include the following
/// packages:
///
/// ```yaml
/// dependencies:
///   # Internationalization support.
///   flutter_localizations:
///     sdk: flutter
///   intl: any # Use the pinned version from flutter_localizations
///
///   # Rest of dependencies
/// ```
///
/// ## iOS Applications
///
/// iOS applications define key application metadata, including supported
/// locales, in an Info.plist file that is built into the application bundle.
/// To configure the locales supported by your app, you’ll need to edit this
/// file.
///
/// First, open your project’s ios/Runner.xcworkspace Xcode workspace file.
/// Then, in the Project Navigator, open the Info.plist file under the Runner
/// project’s Runner folder.
///
/// Next, select the Information Property List item, select Add Item from the
/// Editor menu, then select Localizations from the pop-up menu.
///
/// Select and expand the newly-created Localizations item then, for each
/// locale your application supports, add a new item and select the locale
/// you wish to add from the pop-up menu in the Value field. This list should
/// be consistent with the languages listed in the AppLocalizations.supportedLocales
/// property.
abstract class AppLocalizations {
  AppLocalizations(String locale)
    : localeName = intl.Intl.canonicalizedLocale(locale.toString());

  final String localeName;

  static AppLocalizations of(BuildContext context) {
    return Localizations.of<AppLocalizations>(context, AppLocalizations)!;
  }

  static const LocalizationsDelegate<AppLocalizations> delegate =
      _AppLocalizationsDelegate();

  /// A list of this localizations delegate along with the default localizations
  /// delegates.
  ///
  /// Returns a list of localizations delegates containing this delegate along with
  /// GlobalMaterialLocalizations.delegate, GlobalCupertinoLocalizations.delegate,
  /// and GlobalWidgetsLocalizations.delegate.
  ///
  /// Additional delegates can be added by appending to this list in
  /// MaterialApp. This list does not have to be used at all if a custom list
  /// of delegates is preferred or required.
  static const List<LocalizationsDelegate<dynamic>> localizationsDelegates =
      <LocalizationsDelegate<dynamic>>[
        delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
      ];

  /// A list of this localizations delegate's supported locales.
  static const List<Locale> supportedLocales = <Locale>[
    Locale('ar'),
    Locale('en'),
  ];

  /// No description provided for @appTitle.
  ///
  /// In en, this message translates to:
  /// **'SalehCard'**
  String get appTitle;

  /// No description provided for @emailLabel.
  ///
  /// In en, this message translates to:
  /// **'Email'**
  String get emailLabel;

  /// No description provided for @passwordLabel.
  ///
  /// In en, this message translates to:
  /// **'Password'**
  String get passwordLabel;

  /// No description provided for @signInButton.
  ///
  /// In en, this message translates to:
  /// **'Sign in'**
  String get signInButton;

  /// No description provided for @catalogTitle.
  ///
  /// In en, this message translates to:
  /// **'Products'**
  String get catalogTitle;

  /// No description provided for @retry.
  ///
  /// In en, this message translates to:
  /// **'Retry'**
  String get retry;

  /// No description provided for @logout.
  ///
  /// In en, this message translates to:
  /// **'Log out'**
  String get logout;

  /// No description provided for @outOfStock.
  ///
  /// In en, this message translates to:
  /// **'Out of stock'**
  String get outOfStock;

  /// No description provided for @emptyCatalog.
  ///
  /// In en, this message translates to:
  /// **'No products available.'**
  String get emptyCatalog;

  /// No description provided for @productDetailUnavailable.
  ///
  /// In en, this message translates to:
  /// **'This product is currently unavailable.'**
  String get productDetailUnavailable;

  /// No description provided for @languageToggle.
  ///
  /// In en, this message translates to:
  /// **'العربية'**
  String get languageToggle;

  /// No description provided for @welcomeBackTitle.
  ///
  /// In en, this message translates to:
  /// **'Welcome back'**
  String get welcomeBackTitle;

  /// No description provided for @signInSubtitle.
  ///
  /// In en, this message translates to:
  /// **'Sign in with your mobile number.'**
  String get signInSubtitle;

  /// No description provided for @getStartedTitle.
  ///
  /// In en, this message translates to:
  /// **'Get Started'**
  String get getStartedTitle;

  /// No description provided for @getStartedSubtitle.
  ///
  /// In en, this message translates to:
  /// **'Enter your mobile number. We will send you a confirmation code there.'**
  String get getStartedSubtitle;

  /// No description provided for @createPasswordTitle.
  ///
  /// In en, this message translates to:
  /// **'Create a password'**
  String get createPasswordTitle;

  /// No description provided for @createPasswordSubtitle.
  ///
  /// In en, this message translates to:
  /// **'Choose a strong password to secure your account.'**
  String get createPasswordSubtitle;

  /// No description provided for @verifyTitle.
  ///
  /// In en, this message translates to:
  /// **'Verify your number'**
  String get verifyTitle;

  /// No description provided for @otpSubtitle.
  ///
  /// In en, this message translates to:
  /// **'Enter the 6-digit code we sent to {phone}.'**
  String otpSubtitle(String phone);

  /// No description provided for @continueButton.
  ///
  /// In en, this message translates to:
  /// **'Continue'**
  String get continueButton;

  /// No description provided for @verifyButton.
  ///
  /// In en, this message translates to:
  /// **'Verify'**
  String get verifyButton;

  /// No description provided for @mobileNumberLabel.
  ///
  /// In en, this message translates to:
  /// **'Mobile Number'**
  String get mobileNumberLabel;

  /// No description provided for @phoneHint.
  ///
  /// In en, this message translates to:
  /// **'70 123 456'**
  String get phoneHint;

  /// No description provided for @passwordHint.
  ///
  /// In en, this message translates to:
  /// **'••••••••'**
  String get passwordHint;

  /// No description provided for @newPasswordHint.
  ///
  /// In en, this message translates to:
  /// **'At least 6 characters'**
  String get newPasswordHint;

  /// No description provided for @confirmPasswordLabel.
  ///
  /// In en, this message translates to:
  /// **'Confirm password'**
  String get confirmPasswordLabel;

  /// No description provided for @confirmPasswordHint.
  ///
  /// In en, this message translates to:
  /// **'Re-enter password'**
  String get confirmPasswordHint;

  /// No description provided for @termsPrefix.
  ///
  /// In en, this message translates to:
  /// **'By entering your phone number, you agree to our '**
  String get termsPrefix;

  /// No description provided for @termsLink.
  ///
  /// In en, this message translates to:
  /// **'terms and conditions'**
  String get termsLink;

  /// No description provided for @resendPrefix.
  ///
  /// In en, this message translates to:
  /// **'Didn\'t get a code? '**
  String get resendPrefix;

  /// No description provided for @resendLink.
  ///
  /// In en, this message translates to:
  /// **'Resend'**
  String get resendLink;

  /// No description provided for @haveAccountPrefix.
  ///
  /// In en, this message translates to:
  /// **'Already have an account? '**
  String get haveAccountPrefix;

  /// No description provided for @noAccountPrefix.
  ///
  /// In en, this message translates to:
  /// **'Don\'t have an account? '**
  String get noAccountPrefix;

  /// No description provided for @signUpButton.
  ///
  /// In en, this message translates to:
  /// **'Sign up'**
  String get signUpButton;

  /// No description provided for @invalidPhone.
  ///
  /// In en, this message translates to:
  /// **'Please enter a valid mobile number.'**
  String get invalidPhone;

  /// No description provided for @passwordRequired.
  ///
  /// In en, this message translates to:
  /// **'Please enter your password.'**
  String get passwordRequired;

  /// No description provided for @passwordTooShort.
  ///
  /// In en, this message translates to:
  /// **'Password must be at least 6 characters.'**
  String get passwordTooShort;

  /// No description provided for @passwordMismatch.
  ///
  /// In en, this message translates to:
  /// **'Passwords do not match.'**
  String get passwordMismatch;

  /// No description provided for @otpIncomplete.
  ///
  /// In en, this message translates to:
  /// **'Enter the 6-digit code.'**
  String get otpIncomplete;

  /// No description provided for @searchHint.
  ///
  /// In en, this message translates to:
  /// **'Search'**
  String get searchHint;

  /// No description provided for @totalBalance.
  ///
  /// In en, this message translates to:
  /// **'Total Balance'**
  String get totalBalance;

  /// No description provided for @requestPhysicalCard.
  ///
  /// In en, this message translates to:
  /// **'Request Physical Card'**
  String get requestPhysicalCard;

  /// No description provided for @cardInfo.
  ///
  /// In en, this message translates to:
  /// **'Card info'**
  String get cardInfo;

  /// No description provided for @addMoney.
  ///
  /// In en, this message translates to:
  /// **'Add Money'**
  String get addMoney;

  /// No description provided for @promoTitle.
  ///
  /// In en, this message translates to:
  /// **'Instant digital delivery'**
  String get promoTitle;

  /// No description provided for @promoSubtitle.
  ///
  /// In en, this message translates to:
  /// **'Gift cards & top-ups land in your wallet in seconds.'**
  String get promoSubtitle;

  /// No description provided for @featured.
  ///
  /// In en, this message translates to:
  /// **'Featured'**
  String get featured;

  /// No description provided for @navHome.
  ///
  /// In en, this message translates to:
  /// **'Home'**
  String get navHome;

  /// No description provided for @navCategories.
  ///
  /// In en, this message translates to:
  /// **'Categories'**
  String get navCategories;

  /// No description provided for @navCart.
  ///
  /// In en, this message translates to:
  /// **'Cart'**
  String get navCart;

  /// No description provided for @navMenu.
  ///
  /// In en, this message translates to:
  /// **'Menu'**
  String get navMenu;

  /// No description provided for @comingSoon.
  ///
  /// In en, this message translates to:
  /// **'Coming soon.'**
  String get comingSoon;

  /// No description provided for @loadFailed.
  ///
  /// In en, this message translates to:
  /// **'Couldn\'t load products.'**
  String get loadFailed;

  /// No description provided for @fromLabel.
  ///
  /// In en, this message translates to:
  /// **'from'**
  String get fromLabel;

  /// No description provided for @chooseAmount.
  ///
  /// In en, this message translates to:
  /// **'Choose amount'**
  String get chooseAmount;

  /// No description provided for @quantityLabel.
  ///
  /// In en, this message translates to:
  /// **'Quantity'**
  String get quantityLabel;

  /// No description provided for @requiredBadge.
  ///
  /// In en, this message translates to:
  /// **'Required'**
  String get requiredBadge;

  /// No description provided for @ratingsCount.
  ///
  /// In en, this message translates to:
  /// **'({count} ratings)'**
  String ratingsCount(int count);

  /// No description provided for @enterValue.
  ///
  /// In en, this message translates to:
  /// **'Enter {field}'**
  String enterValue(String field);

  /// No description provided for @deliveredInstantly.
  ///
  /// In en, this message translates to:
  /// **'Credit is delivered to this account instantly.'**
  String get deliveredInstantly;

  /// No description provided for @addToCart.
  ///
  /// In en, this message translates to:
  /// **'Add to cart'**
  String get addToCart;

  /// No description provided for @buyNow.
  ///
  /// In en, this message translates to:
  /// **'Buy now'**
  String get buyNow;

  /// No description provided for @notifyMe.
  ///
  /// In en, this message translates to:
  /// **'Notify me'**
  String get notifyMe;

  /// No description provided for @totalLabel.
  ///
  /// In en, this message translates to:
  /// **'Total'**
  String get totalLabel;

  /// No description provided for @cartTitle.
  ///
  /// In en, this message translates to:
  /// **'Your cart'**
  String get cartTitle;

  /// No description provided for @cartItemsCount.
  ///
  /// In en, this message translates to:
  /// **'· {count} items'**
  String cartItemsCount(int count);

  /// No description provided for @cartEmptyTitle.
  ///
  /// In en, this message translates to:
  /// **'Your cart is empty'**
  String get cartEmptyTitle;

  /// No description provided for @cartEmptySub.
  ///
  /// In en, this message translates to:
  /// **'Browse the catalog and add gift cards & top-ups.'**
  String get cartEmptySub;

  /// No description provided for @browseCatalog.
  ///
  /// In en, this message translates to:
  /// **'Browse catalog'**
  String get browseCatalog;

  /// No description provided for @subtotalLabel.
  ///
  /// In en, this message translates to:
  /// **'Subtotal'**
  String get subtotalLabel;

  /// No description provided for @discountLabel.
  ///
  /// In en, this message translates to:
  /// **'Discount'**
  String get discountLabel;

  /// No description provided for @walletBalanceHint.
  ///
  /// In en, this message translates to:
  /// **'Wallet balance:'**
  String get walletBalanceHint;

  /// No description provided for @checkoutCta.
  ///
  /// In en, this message translates to:
  /// **'Checkout'**
  String get checkoutCta;

  /// No description provided for @checkoutTitle.
  ///
  /// In en, this message translates to:
  /// **'Checkout'**
  String get checkoutTitle;

  /// No description provided for @paymentFailed.
  ///
  /// In en, this message translates to:
  /// **'Payment failed. Please try another method.'**
  String get paymentFailed;

  /// No description provided for @deliveryDetails.
  ///
  /// In en, this message translates to:
  /// **'Delivery details'**
  String get deliveryDetails;

  /// No description provided for @paymentMethodLabel.
  ///
  /// In en, this message translates to:
  /// **'Payment method'**
  String get paymentMethodLabel;

  /// No description provided for @payCardTitle.
  ///
  /// In en, this message translates to:
  /// **'Card'**
  String get payCardTitle;

  /// No description provided for @payCardSub.
  ///
  /// In en, this message translates to:
  /// **'Instant approval'**
  String get payCardSub;

  /// No description provided for @payWalletTitle.
  ///
  /// In en, this message translates to:
  /// **'Wallet'**
  String get payWalletTitle;

  /// No description provided for @balanceLabel.
  ///
  /// In en, this message translates to:
  /// **'Balance'**
  String get balanceLabel;

  /// No description provided for @insufficientBalance.
  ///
  /// In en, this message translates to:
  /// **'Insufficient balance'**
  String get insufficientBalance;

  /// No description provided for @payUsdtTitle.
  ///
  /// In en, this message translates to:
  /// **'USDT'**
  String get payUsdtTitle;

  /// No description provided for @payUsdtSub.
  ///
  /// In en, this message translates to:
  /// **'Auto-approve · Instant'**
  String get payUsdtSub;

  /// No description provided for @promoCodePlaceholder.
  ///
  /// In en, this message translates to:
  /// **'Promo code'**
  String get promoCodePlaceholder;

  /// No description provided for @applyLabel.
  ///
  /// In en, this message translates to:
  /// **'Apply'**
  String get applyLabel;

  /// No description provided for @placeOrderCta.
  ///
  /// In en, this message translates to:
  /// **'Place order'**
  String get placeOrderCta;

  /// No description provided for @fieldRequired.
  ///
  /// In en, this message translates to:
  /// **'This field is required.'**
  String get fieldRequired;

  /// No description provided for @orderCompletedTitle.
  ///
  /// In en, this message translates to:
  /// **'Order completed'**
  String get orderCompletedTitle;

  /// No description provided for @orderProcessingTitle.
  ///
  /// In en, this message translates to:
  /// **'We\'re completing your order'**
  String get orderProcessingTitle;

  /// No description provided for @orderCompletedSub.
  ///
  /// In en, this message translates to:
  /// **'Your codes have been delivered. Enjoy!'**
  String get orderCompletedSub;

  /// No description provided for @orderProcessingSub.
  ///
  /// In en, this message translates to:
  /// **'Account-credit orders can take a few minutes.'**
  String get orderProcessingSub;

  /// No description provided for @orderLabel.
  ///
  /// In en, this message translates to:
  /// **'Order'**
  String get orderLabel;

  /// No description provided for @deliveredCodesLabel.
  ///
  /// In en, this message translates to:
  /// **'Delivered codes'**
  String get deliveredCodesLabel;

  /// No description provided for @giftCardLabel.
  ///
  /// In en, this message translates to:
  /// **'Gift card'**
  String get giftCardLabel;

  /// No description provided for @copyLabel.
  ///
  /// In en, this message translates to:
  /// **'Copy'**
  String get copyLabel;

  /// No description provided for @copiedLabel.
  ///
  /// In en, this message translates to:
  /// **'Copied'**
  String get copiedLabel;

  /// No description provided for @orderProcessingNote.
  ///
  /// In en, this message translates to:
  /// **'We\'ll notify you the moment it\'s done. You can track its status from the order page.'**
  String get orderProcessingNote;

  /// No description provided for @viewOrderCta.
  ///
  /// In en, this message translates to:
  /// **'View order'**
  String get viewOrderCta;

  /// No description provided for @backToHomeCta.
  ///
  /// In en, this message translates to:
  /// **'Back to home'**
  String get backToHomeCta;
}

class _AppLocalizationsDelegate
    extends LocalizationsDelegate<AppLocalizations> {
  const _AppLocalizationsDelegate();

  @override
  Future<AppLocalizations> load(Locale locale) {
    return SynchronousFuture<AppLocalizations>(lookupAppLocalizations(locale));
  }

  @override
  bool isSupported(Locale locale) =>
      <String>['ar', 'en'].contains(locale.languageCode);

  @override
  bool shouldReload(_AppLocalizationsDelegate old) => false;
}

AppLocalizations lookupAppLocalizations(Locale locale) {
  // Lookup logic when only language code is specified.
  switch (locale.languageCode) {
    case 'ar':
      return AppLocalizationsAr();
    case 'en':
      return AppLocalizationsEn();
  }

  throw FlutterError(
    'AppLocalizations.delegate failed to load unsupported locale "$locale". This is likely '
    'an issue with the localizations generation tool. Please file an issue '
    'on GitHub with a reproducible sample app and the gen-l10n configuration '
    'that was used.',
  );
}
