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

  /// No description provided for @sendCodeButton.
  ///
  /// In en, this message translates to:
  /// **'Send code'**
  String get sendCodeButton;

  /// No description provided for @useCodeInstead.
  ///
  /// In en, this message translates to:
  /// **'Sign in with a code instead'**
  String get useCodeInstead;

  /// No description provided for @usePasswordInstead.
  ///
  /// In en, this message translates to:
  /// **'Sign in with a password instead'**
  String get usePasswordInstead;

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

  /// No description provided for @offersEmpty.
  ///
  /// In en, this message translates to:
  /// **'No offers right now. Check back soon.'**
  String get offersEmpty;

  /// No description provided for @productDetailUnavailable.
  ///
  /// In en, this message translates to:
  /// **'This product is currently unavailable.'**
  String get productDetailUnavailable;

  /// No description provided for @verifyChecking.
  ///
  /// In en, this message translates to:
  /// **'Checking ID…'**
  String get verifyChecking;

  /// No description provided for @verifyFound.
  ///
  /// In en, this message translates to:
  /// **'Account'**
  String get verifyFound;

  /// No description provided for @verifyNotFound.
  ///
  /// In en, this message translates to:
  /// **'We couldn\'t find that ID. Please check and try again.'**
  String get verifyNotFound;

  /// No description provided for @verifyUnavailable.
  ///
  /// In en, this message translates to:
  /// **'Couldn\'t verify right now — you can still continue.'**
  String get verifyUnavailable;

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

  /// No description provided for @nameLabel.
  ///
  /// In en, this message translates to:
  /// **'Full name'**
  String get nameLabel;

  /// No description provided for @nameHint.
  ///
  /// In en, this message translates to:
  /// **'Your name'**
  String get nameHint;

  /// No description provided for @nameRequired.
  ///
  /// In en, this message translates to:
  /// **'Please enter your name.'**
  String get nameRequired;

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
  /// **'At least 8 characters'**
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
  /// **'Password must be at least 8 characters.'**
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
  /// **'Gift card codes delivered instantly after checkout.'**
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

  /// No description provided for @navOffers.
  ///
  /// In en, this message translates to:
  /// **'Offers'**
  String get navOffers;

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

  /// No description provided for @qtyTotalLine.
  ///
  /// In en, this message translates to:
  /// **'{count} × {unitPrice} = {total}'**
  String qtyTotalLine(int count, String unitPrice, String total);

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

  /// No description provided for @creditAfterProcessing.
  ///
  /// In en, this message translates to:
  /// **'Credit is added to this account after we process your order.'**
  String get creditAfterProcessing;

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
  /// **'· {count, plural, =1{1 item} other{{count} items}}'**
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

  /// No description provided for @topUpWalletCta.
  ///
  /// In en, this message translates to:
  /// **'Top up wallet'**
  String get topUpWalletCta;

  /// No description provided for @kycRequiredTitle.
  ///
  /// In en, this message translates to:
  /// **'Verification required'**
  String get kycRequiredTitle;

  /// No description provided for @kycRequiredBody.
  ///
  /// In en, this message translates to:
  /// **'Verify your identity to place orders. It only takes a minute and is reviewed by our team.'**
  String get kycRequiredBody;

  /// No description provided for @kycRequiredCta.
  ///
  /// In en, this message translates to:
  /// **'Verify identity'**
  String get kycRequiredCta;

  /// No description provided for @kycBannerBody.
  ///
  /// In en, this message translates to:
  /// **'Verify your account to make purchases — it only takes a minute. Tap to start.'**
  String get kycBannerBody;

  /// No description provided for @kycBannerPending.
  ///
  /// In en, this message translates to:
  /// **'Your verification is being reviewed. You can purchase once it\'s approved.'**
  String get kycBannerPending;

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

  /// No description provided for @invalidLebanesePhone.
  ///
  /// In en, this message translates to:
  /// **'Enter a valid Lebanese mobile number (e.g. 71 123 456).'**
  String get invalidLebanesePhone;

  /// No description provided for @amountNotANumber.
  ///
  /// In en, this message translates to:
  /// **'Enter a valid number.'**
  String get amountNotANumber;

  /// No description provided for @amountOutOfRange.
  ///
  /// In en, this message translates to:
  /// **'Enter an amount between {min} and {max}.'**
  String amountOutOfRange(String min, String max);

  /// No description provided for @amountAtLeast.
  ///
  /// In en, this message translates to:
  /// **'Enter an amount of at least {min}.'**
  String amountAtLeast(String min);

  /// No description provided for @amountAtMost.
  ///
  /// In en, this message translates to:
  /// **'Enter an amount of at most {max}.'**
  String amountAtMost(String max);

  /// No description provided for @amountRangeHint.
  ///
  /// In en, this message translates to:
  /// **'Between {min} and {max}'**
  String amountRangeHint(String min, String max);

  /// No description provided for @amountMinHint.
  ///
  /// In en, this message translates to:
  /// **'Minimum {min}'**
  String amountMinHint(String min);

  /// No description provided for @amountMaxHint.
  ///
  /// In en, this message translates to:
  /// **'Maximum {max}'**
  String amountMaxHint(String max);

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

  /// No description provided for @orderItemsLabel.
  ///
  /// In en, this message translates to:
  /// **'Items'**
  String get orderItemsLabel;

  /// No description provided for @statusPending.
  ///
  /// In en, this message translates to:
  /// **'Pending'**
  String get statusPending;

  /// No description provided for @statusProcessing.
  ///
  /// In en, this message translates to:
  /// **'Processing'**
  String get statusProcessing;

  /// No description provided for @statusCompleted.
  ///
  /// In en, this message translates to:
  /// **'Completed'**
  String get statusCompleted;

  /// No description provided for @statusFailed.
  ///
  /// In en, this message translates to:
  /// **'Failed'**
  String get statusFailed;

  /// No description provided for @statusRefunded.
  ///
  /// In en, this message translates to:
  /// **'Refunded'**
  String get statusRefunded;

  /// No description provided for @ordersTitle.
  ///
  /// In en, this message translates to:
  /// **'Orders'**
  String get ordersTitle;

  /// No description provided for @filterAll.
  ///
  /// In en, this message translates to:
  /// **'All'**
  String get filterAll;

  /// No description provided for @ordersEmptyTitle.
  ///
  /// In en, this message translates to:
  /// **'No orders yet'**
  String get ordersEmptyTitle;

  /// No description provided for @ordersEmptySub.
  ///
  /// In en, this message translates to:
  /// **'Your purchases will show up here once you place an order.'**
  String get ordersEmptySub;

  /// No description provided for @orderRefundedNote.
  ///
  /// In en, this message translates to:
  /// **'This order was refunded. The amount was returned to your wallet.'**
  String get orderRefundedNote;

  /// No description provided for @orderTimelineLabel.
  ///
  /// In en, this message translates to:
  /// **'Timeline'**
  String get orderTimelineLabel;

  /// No description provided for @walletTitle.
  ///
  /// In en, this message translates to:
  /// **'Wallet'**
  String get walletTitle;

  /// No description provided for @currentBalance.
  ///
  /// In en, this message translates to:
  /// **'Current balance'**
  String get currentBalance;

  /// No description provided for @topUpCta.
  ///
  /// In en, this message translates to:
  /// **'Top up'**
  String get topUpCta;

  /// No description provided for @sendMoney.
  ///
  /// In en, this message translates to:
  /// **'Send money'**
  String get sendMoney;

  /// No description provided for @txHistory.
  ///
  /// In en, this message translates to:
  /// **'Transaction history'**
  String get txHistory;

  /// No description provided for @walletEmptyTitle.
  ///
  /// In en, this message translates to:
  /// **'No transactions yet'**
  String get walletEmptyTitle;

  /// No description provided for @walletEmptySub.
  ///
  /// In en, this message translates to:
  /// **'Top up your wallet to get started — your activity will show up here.'**
  String get walletEmptySub;

  /// No description provided for @txTopUp.
  ///
  /// In en, this message translates to:
  /// **'Top up'**
  String get txTopUp;

  /// No description provided for @txPurchase.
  ///
  /// In en, this message translates to:
  /// **'Purchase'**
  String get txPurchase;

  /// No description provided for @txRefund.
  ///
  /// In en, this message translates to:
  /// **'Refund'**
  String get txRefund;

  /// No description provided for @txAdjustment.
  ///
  /// In en, this message translates to:
  /// **'Adjustment'**
  String get txAdjustment;

  /// No description provided for @topUpAmount.
  ///
  /// In en, this message translates to:
  /// **'Top-up amount'**
  String get topUpAmount;

  /// No description provided for @topUpVia.
  ///
  /// In en, this message translates to:
  /// **'How will you pay?'**
  String get topUpVia;

  /// No description provided for @topUpNoteHint.
  ///
  /// In en, this message translates to:
  /// **'Payment reference / note (optional)'**
  String get topUpNoteHint;

  /// No description provided for @topUpRequested.
  ///
  /// In en, this message translates to:
  /// **'Request submitted — your wallet is credited once an admin confirms your payment.'**
  String get topUpRequested;

  /// No description provided for @topUpRequestsTitle.
  ///
  /// In en, this message translates to:
  /// **'My top-up requests'**
  String get topUpRequestsTitle;

  /// No description provided for @statusApproved.
  ///
  /// In en, this message translates to:
  /// **'Approved'**
  String get statusApproved;

  /// No description provided for @statusRejected.
  ///
  /// In en, this message translates to:
  /// **'Rejected'**
  String get statusRejected;

  /// No description provided for @walletPendingTitle.
  ///
  /// In en, this message translates to:
  /// **'{count, plural, =1{Top-up pending approval} other{{count} top-ups pending approval}}'**
  String walletPendingTitle(int count);

  /// No description provided for @walletPendingHint.
  ///
  /// In en, this message translates to:
  /// **'Balance updates once an admin confirms your payment.'**
  String get walletPendingHint;

  /// No description provided for @sendMoneyComingSoon.
  ///
  /// In en, this message translates to:
  /// **'Send money is coming soon.'**
  String get sendMoneyComingSoon;

  /// No description provided for @recipientLabel.
  ///
  /// In en, this message translates to:
  /// **'Recipient (phone or email)'**
  String get recipientLabel;

  /// No description provided for @amountLabel.
  ///
  /// In en, this message translates to:
  /// **'Amount'**
  String get amountLabel;

  /// No description provided for @noteLabel.
  ///
  /// In en, this message translates to:
  /// **'Note (optional)'**
  String get noteLabel;

  /// No description provided for @sendCta.
  ///
  /// In en, this message translates to:
  /// **'Send'**
  String get sendCta;

  /// No description provided for @darkMode.
  ///
  /// In en, this message translates to:
  /// **'Dark mode'**
  String get darkMode;

  /// No description provided for @profileTitle.
  ///
  /// In en, this message translates to:
  /// **'Profile'**
  String get profileTitle;

  /// No description provided for @accountInfo.
  ///
  /// In en, this message translates to:
  /// **'Account info'**
  String get accountInfo;

  /// No description provided for @roleLabel.
  ///
  /// In en, this message translates to:
  /// **'Role'**
  String get roleLabel;

  /// No description provided for @loyaltyPoints.
  ///
  /// In en, this message translates to:
  /// **'Loyalty points'**
  String get loyaltyPoints;

  /// No description provided for @verificationTitle.
  ///
  /// In en, this message translates to:
  /// **'Verification'**
  String get verificationTitle;

  /// No description provided for @savedPlayerIds.
  ///
  /// In en, this message translates to:
  /// **'Saved player IDs'**
  String get savedPlayerIds;

  /// No description provided for @savedPlayerIdsEmptyTitle.
  ///
  /// In en, this message translates to:
  /// **'No saved player IDs'**
  String get savedPlayerIdsEmptyTitle;

  /// No description provided for @savedPlayerIdsEmptySub.
  ///
  /// In en, this message translates to:
  /// **'Save your player or account IDs for faster checkout.'**
  String get savedPlayerIdsEmptySub;

  /// No description provided for @addPlayerId.
  ///
  /// In en, this message translates to:
  /// **'Add'**
  String get addPlayerId;

  /// No description provided for @playerIdLabelHint.
  ///
  /// In en, this message translates to:
  /// **'Label (e.g. PUBG main)'**
  String get playerIdLabelHint;

  /// No description provided for @playerIdHint.
  ///
  /// In en, this message translates to:
  /// **'Enter a player ID'**
  String get playerIdHint;

  /// No description provided for @editProfile.
  ///
  /// In en, this message translates to:
  /// **'Edit'**
  String get editProfile;

  /// No description provided for @cancel.
  ///
  /// In en, this message translates to:
  /// **'Cancel'**
  String get cancel;

  /// No description provided for @saveChanges.
  ///
  /// In en, this message translates to:
  /// **'Save changes'**
  String get saveChanges;

  /// No description provided for @profileSaved.
  ///
  /// In en, this message translates to:
  /// **'Profile updated.'**
  String get profileSaved;

  /// No description provided for @browseTitle.
  ///
  /// In en, this message translates to:
  /// **'Categories'**
  String get browseTitle;

  /// No description provided for @categoryItemCount.
  ///
  /// In en, this message translates to:
  /// **'{count, plural, =0{No items} =1{1 item} other{{count} items}}'**
  String categoryItemCount(int count);

  /// No description provided for @searchPromptTitle.
  ///
  /// In en, this message translates to:
  /// **'Search the catalog'**
  String get searchPromptTitle;

  /// No description provided for @searchPromptSub.
  ///
  /// In en, this message translates to:
  /// **'Type to find gift cards, top-ups & more.'**
  String get searchPromptSub;

  /// No description provided for @searchEmptyTitle.
  ///
  /// In en, this message translates to:
  /// **'No results'**
  String get searchEmptyTitle;

  /// No description provided for @searchEmptySub.
  ///
  /// In en, this message translates to:
  /// **'Try a different keyword or browse categories.'**
  String get searchEmptySub;

  /// No description provided for @notificationsTitle.
  ///
  /// In en, this message translates to:
  /// **'Notifications'**
  String get notificationsTitle;

  /// No description provided for @notificationsEmptyTitle.
  ///
  /// In en, this message translates to:
  /// **'You\'re all caught up'**
  String get notificationsEmptyTitle;

  /// No description provided for @notificationsEmptySub.
  ///
  /// In en, this message translates to:
  /// **'Notifications about your orders and wallet will show up here.'**
  String get notificationsEmptySub;

  /// No description provided for @notifOrderCompletedTitle.
  ///
  /// In en, this message translates to:
  /// **'Order completed'**
  String get notifOrderCompletedTitle;

  /// No description provided for @notifOrderCompletedBody.
  ///
  /// In en, this message translates to:
  /// **'Your order was delivered — {amount}.'**
  String notifOrderCompletedBody(String amount);

  /// No description provided for @notifOrderRefundedTitle.
  ///
  /// In en, this message translates to:
  /// **'Order refunded'**
  String get notifOrderRefundedTitle;

  /// No description provided for @notifOrderRefundedBody.
  ///
  /// In en, this message translates to:
  /// **'{amount} was returned to your wallet.'**
  String notifOrderRefundedBody(String amount);

  /// No description provided for @notifWalletTopUpTitle.
  ///
  /// In en, this message translates to:
  /// **'Wallet topped up'**
  String get notifWalletTopUpTitle;

  /// No description provided for @notifWalletTopUpBody.
  ///
  /// In en, this message translates to:
  /// **'Your top-up of {amount} was approved and added to your wallet.'**
  String notifWalletTopUpBody(String amount);

  /// No description provided for @notifTopUpRejectedTitle.
  ///
  /// In en, this message translates to:
  /// **'Top-up rejected'**
  String get notifTopUpRejectedTitle;

  /// No description provided for @notifTopUpRejectedBody.
  ///
  /// In en, this message translates to:
  /// **'Your {amount} top-up request was rejected.'**
  String notifTopUpRejectedBody(String amount);

  /// No description provided for @notifKycApprovedTitle.
  ///
  /// In en, this message translates to:
  /// **'Identity verified'**
  String get notifKycApprovedTitle;

  /// No description provided for @notifKycApprovedBody.
  ///
  /// In en, this message translates to:
  /// **'Your account is verified — you can now purchase.'**
  String get notifKycApprovedBody;

  /// No description provided for @notifKycRejectedTitle.
  ///
  /// In en, this message translates to:
  /// **'Verification rejected'**
  String get notifKycRejectedTitle;

  /// No description provided for @notifKycRejectedBody.
  ///
  /// In en, this message translates to:
  /// **'Please resubmit your information.'**
  String get notifKycRejectedBody;

  /// No description provided for @notifPromoTitle.
  ///
  /// In en, this message translates to:
  /// **'Limited-time offer'**
  String get notifPromoTitle;

  /// No description provided for @notifPromoBody.
  ///
  /// In en, this message translates to:
  /// **'Discover the latest gift cards and offers.'**
  String get notifPromoBody;

  /// No description provided for @kycTitle.
  ///
  /// In en, this message translates to:
  /// **'Verification'**
  String get kycTitle;

  /// No description provided for @kycMenuLabel.
  ///
  /// In en, this message translates to:
  /// **'Verification'**
  String get kycMenuLabel;

  /// No description provided for @kycBadgeUnverified.
  ///
  /// In en, this message translates to:
  /// **'Not verified'**
  String get kycBadgeUnverified;

  /// No description provided for @kycBadgePending.
  ///
  /// In en, this message translates to:
  /// **'Under review'**
  String get kycBadgePending;

  /// No description provided for @kycBadgeVerified.
  ///
  /// In en, this message translates to:
  /// **'Verified'**
  String get kycBadgeVerified;

  /// No description provided for @kycBadgeRejected.
  ///
  /// In en, this message translates to:
  /// **'Rejected'**
  String get kycBadgeRejected;

  /// No description provided for @kycUnverifiedTitle.
  ///
  /// In en, this message translates to:
  /// **'Verify your identity'**
  String get kycUnverifiedTitle;

  /// No description provided for @kycUnverifiedBody.
  ///
  /// In en, this message translates to:
  /// **'Verify your identity to unlock higher limits and faster checkout. It only takes a minute.'**
  String get kycUnverifiedBody;

  /// No description provided for @kycPendingTitle.
  ///
  /// In en, this message translates to:
  /// **'Under review'**
  String get kycPendingTitle;

  /// No description provided for @kycPendingBody.
  ///
  /// In en, this message translates to:
  /// **'We\'ve received your documents and our team is reviewing them. This usually takes a few minutes.'**
  String get kycPendingBody;

  /// No description provided for @kycVerifiedTitle.
  ///
  /// In en, this message translates to:
  /// **'You\'re verified'**
  String get kycVerifiedTitle;

  /// No description provided for @kycVerifiedBody.
  ///
  /// In en, this message translates to:
  /// **'Your identity has been confirmed. All features are unlocked — enjoy higher limits and faster checkout.'**
  String get kycVerifiedBody;

  /// No description provided for @kycRejectedTitle.
  ///
  /// In en, this message translates to:
  /// **'Verification failed'**
  String get kycRejectedTitle;

  /// No description provided for @kycRejectedBody.
  ///
  /// In en, this message translates to:
  /// **'We couldn\'t verify your identity from the documents provided. Please check the details and resubmit.'**
  String get kycRejectedBody;

  /// No description provided for @kycVerifyNowCta.
  ///
  /// In en, this message translates to:
  /// **'Verify now'**
  String get kycVerifyNowCta;

  /// No description provided for @kycResubmitCta.
  ///
  /// In en, this message translates to:
  /// **'Resubmit'**
  String get kycResubmitCta;

  /// No description provided for @kycFormTitle.
  ///
  /// In en, this message translates to:
  /// **'Identity verification'**
  String get kycFormTitle;

  /// No description provided for @kycFullNameLabel.
  ///
  /// In en, this message translates to:
  /// **'Full name'**
  String get kycFullNameLabel;

  /// No description provided for @kycDobLabel.
  ///
  /// In en, this message translates to:
  /// **'Date of birth'**
  String get kycDobLabel;

  /// No description provided for @kycDobHint.
  ///
  /// In en, this message translates to:
  /// **'Select your date of birth'**
  String get kycDobHint;

  /// No description provided for @kycPlaceOfBirthLabel.
  ///
  /// In en, this message translates to:
  /// **'Place of birth'**
  String get kycPlaceOfBirthLabel;

  /// No description provided for @kycPlaceOfResidenceLabel.
  ///
  /// In en, this message translates to:
  /// **'Place of residence'**
  String get kycPlaceOfResidenceLabel;

  /// No description provided for @kycPlaceHint.
  ///
  /// In en, this message translates to:
  /// **'City, country'**
  String get kycPlaceHint;

  /// No description provided for @kycDocTypeLabel.
  ///
  /// In en, this message translates to:
  /// **'Document type'**
  String get kycDocTypeLabel;

  /// No description provided for @kycDocPassport.
  ///
  /// In en, this message translates to:
  /// **'Passport'**
  String get kycDocPassport;

  /// No description provided for @kycDocIdCard.
  ///
  /// In en, this message translates to:
  /// **'ID card'**
  String get kycDocIdCard;

  /// No description provided for @kycDocLicense.
  ///
  /// In en, this message translates to:
  /// **'Driver\'s license'**
  String get kycDocLicense;

  /// No description provided for @kycDocNumberLabel.
  ///
  /// In en, this message translates to:
  /// **'Document number'**
  String get kycDocNumberLabel;

  /// No description provided for @kycSubmitCta.
  ///
  /// In en, this message translates to:
  /// **'Submit'**
  String get kycSubmitCta;

  /// No description provided for @kycFieldRequired.
  ///
  /// In en, this message translates to:
  /// **'This field is required.'**
  String get kycFieldRequired;

  /// No description provided for @kycSubmittedSnack.
  ///
  /// In en, this message translates to:
  /// **'Verification submitted — we\'ll review it shortly.'**
  String get kycSubmittedSnack;

  /// No description provided for @kycDocPhotosLabel.
  ///
  /// In en, this message translates to:
  /// **'Document photos'**
  String get kycDocPhotosLabel;

  /// No description provided for @kycDocFrontLabel.
  ///
  /// In en, this message translates to:
  /// **'Front of document'**
  String get kycDocFrontLabel;

  /// No description provided for @kycDocBackLabel.
  ///
  /// In en, this message translates to:
  /// **'Back of document'**
  String get kycDocBackLabel;

  /// No description provided for @kycDocBackOptionalTag.
  ///
  /// In en, this message translates to:
  /// **'Optional for passports'**
  String get kycDocBackOptionalTag;

  /// No description provided for @kycDocAddPhoto.
  ///
  /// In en, this message translates to:
  /// **'Add photo'**
  String get kycDocAddPhoto;

  /// No description provided for @kycDocReplacePhoto.
  ///
  /// In en, this message translates to:
  /// **'Replace photo'**
  String get kycDocReplacePhoto;

  /// No description provided for @kycDocRemovePhoto.
  ///
  /// In en, this message translates to:
  /// **'Remove'**
  String get kycDocRemovePhoto;

  /// No description provided for @kycDocUploading.
  ///
  /// In en, this message translates to:
  /// **'Uploading…'**
  String get kycDocUploading;

  /// No description provided for @kycDocUploadFailed.
  ///
  /// In en, this message translates to:
  /// **'Upload failed — tap to retry.'**
  String get kycDocUploadFailed;

  /// No description provided for @kycDocPhotoRequired.
  ///
  /// In en, this message translates to:
  /// **'This photo is required.'**
  String get kycDocPhotoRequired;

  /// No description provided for @kycDocSourceCamera.
  ///
  /// In en, this message translates to:
  /// **'Take a photo'**
  String get kycDocSourceCamera;

  /// No description provided for @kycDocSourceGallery.
  ///
  /// In en, this message translates to:
  /// **'Choose from gallery'**
  String get kycDocSourceGallery;

  /// No description provided for @writeReview.
  ///
  /// In en, this message translates to:
  /// **'Write a review'**
  String get writeReview;

  /// No description provided for @reviewSheetTitle.
  ///
  /// In en, this message translates to:
  /// **'Rate this product'**
  String get reviewSheetTitle;

  /// No description provided for @reviewTapToRate.
  ///
  /// In en, this message translates to:
  /// **'Tap a star to rate'**
  String get reviewTapToRate;

  /// No description provided for @reviewNoteOptionalHint.
  ///
  /// In en, this message translates to:
  /// **'Tell us what went wrong (optional)'**
  String get reviewNoteOptionalHint;

  /// No description provided for @reviewSubmit.
  ///
  /// In en, this message translates to:
  /// **'Submit review'**
  String get reviewSubmit;

  /// No description provided for @reviewSubmittedPending.
  ///
  /// In en, this message translates to:
  /// **'Thanks! Your review is pending approval.'**
  String get reviewSubmittedPending;

  /// No description provided for @reviewAlreadyReviewed.
  ///
  /// In en, this message translates to:
  /// **'You\'ve already reviewed this product.'**
  String get reviewAlreadyReviewed;

  /// No description provided for @reviewYouReviewed.
  ///
  /// In en, this message translates to:
  /// **'You reviewed this product'**
  String get reviewYouReviewed;

  /// No description provided for @commonDone.
  ///
  /// In en, this message translates to:
  /// **'Done'**
  String get commonDone;

  /// No description provided for @usdtPayLabel.
  ///
  /// In en, this message translates to:
  /// **'Pay with USDT (crypto)'**
  String get usdtPayLabel;

  /// No description provided for @usdtDepositTitle.
  ///
  /// In en, this message translates to:
  /// **'Send USDT'**
  String get usdtDepositTitle;

  /// No description provided for @usdtSendExactly.
  ///
  /// In en, this message translates to:
  /// **'Send exactly this amount — every digit matters, or your payment can\'t be matched automatically:'**
  String get usdtSendExactly;

  /// No description provided for @usdtAmountLabel.
  ///
  /// In en, this message translates to:
  /// **'Amount'**
  String get usdtAmountLabel;

  /// No description provided for @usdtAmountCopied.
  ///
  /// In en, this message translates to:
  /// **'Amount copied'**
  String get usdtAmountCopied;

  /// No description provided for @usdtAddressLabel.
  ///
  /// In en, this message translates to:
  /// **'{network} deposit address'**
  String usdtAddressLabel(String network);

  /// No description provided for @usdtAddressCopied.
  ///
  /// In en, this message translates to:
  /// **'Address copied'**
  String get usdtAddressCopied;

  /// No description provided for @usdtNetworkWarning.
  ///
  /// In en, this message translates to:
  /// **'Send only USDT on the {network} network. Sending any other coin or network will lose the funds.'**
  String usdtNetworkWarning(String network);

  /// No description provided for @usdtExpiresIn.
  ///
  /// In en, this message translates to:
  /// **'Expires in {time}'**
  String usdtExpiresIn(String time);

  /// No description provided for @usdtWaiting.
  ///
  /// In en, this message translates to:
  /// **'Waiting for your payment…'**
  String get usdtWaiting;

  /// No description provided for @usdtConfirming.
  ///
  /// In en, this message translates to:
  /// **'Payment detected — confirming on-chain…'**
  String get usdtConfirming;

  /// No description provided for @usdtConfirmedTitle.
  ///
  /// In en, this message translates to:
  /// **'Payment confirmed'**
  String get usdtConfirmedTitle;

  /// No description provided for @usdtConfirmedBody.
  ///
  /// In en, this message translates to:
  /// **'Your USDT payment was confirmed.'**
  String get usdtConfirmedBody;

  /// No description provided for @usdtExpiredTitle.
  ///
  /// In en, this message translates to:
  /// **'Payment window expired'**
  String get usdtExpiredTitle;

  /// No description provided for @usdtExpiredBody.
  ///
  /// In en, this message translates to:
  /// **'No payment was received in time. Start again to get a fresh address.'**
  String get usdtExpiredBody;

  /// No description provided for @usdtNetworkLabel.
  ///
  /// In en, this message translates to:
  /// **'Network'**
  String get usdtNetworkLabel;

  /// No description provided for @privacyPolicyMenuLabel.
  ///
  /// In en, this message translates to:
  /// **'Privacy policy'**
  String get privacyPolicyMenuLabel;

  /// No description provided for @deleteAccountMenuLabel.
  ///
  /// In en, this message translates to:
  /// **'Delete account'**
  String get deleteAccountMenuLabel;

  /// No description provided for @deleteAccountTitle.
  ///
  /// In en, this message translates to:
  /// **'Delete your account?'**
  String get deleteAccountTitle;

  /// No description provided for @deleteAccountWarnPermanent.
  ///
  /// In en, this message translates to:
  /// **'This is permanent — your account can\'t be recovered, and you\'ll be signed out on all devices.'**
  String get deleteAccountWarnPermanent;

  /// No description provided for @deleteAccountWarnKycDeleted.
  ///
  /// In en, this message translates to:
  /// **'Your identity (ID) photos are permanently deleted.'**
  String get deleteAccountWarnKycDeleted;

  /// No description provided for @deleteAccountWarnOrdersKept.
  ///
  /// In en, this message translates to:
  /// **'Past orders are kept anonymized, as required for accounting.'**
  String get deleteAccountWarnOrdersKept;

  /// No description provided for @deleteAccountWarnWalletEmpty.
  ///
  /// In en, this message translates to:
  /// **'Your wallet must be empty before you can delete your account.'**
  String get deleteAccountWarnWalletEmpty;

  /// No description provided for @deleteAccountConfirmPrompt.
  ///
  /// In en, this message translates to:
  /// **'To confirm, type \"{word}\" below.'**
  String deleteAccountConfirmPrompt(String word);

  /// No description provided for @deleteAccountConfirmWord.
  ///
  /// In en, this message translates to:
  /// **'DELETE'**
  String get deleteAccountConfirmWord;

  /// No description provided for @deleteAccountCta.
  ///
  /// In en, this message translates to:
  /// **'Delete my account'**
  String get deleteAccountCta;

  /// No description provided for @deleteAccountWalletNotEmpty.
  ///
  /// In en, this message translates to:
  /// **'Your wallet still has a balance. Spend or use it first, then try again.'**
  String get deleteAccountWalletNotEmpty;

  /// No description provided for @deleteAccountOrdersInFlight.
  ///
  /// In en, this message translates to:
  /// **'You have orders still being processed. Please wait until they finish, then try again.'**
  String get deleteAccountOrdersInFlight;

  /// No description provided for @deleteAccountPaymentsPending.
  ///
  /// In en, this message translates to:
  /// **'You have a payment in progress. Please wait until it completes or expires, then try again.'**
  String get deleteAccountPaymentsPending;

  /// No description provided for @deleteAccountTopUpsPending.
  ///
  /// In en, this message translates to:
  /// **'You have a wallet top-up awaiting approval. Please wait for its decision, then try again.'**
  String get deleteAccountTopUpsPending;
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
