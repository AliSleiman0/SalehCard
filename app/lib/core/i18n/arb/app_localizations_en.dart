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

  @override
  String get searchHint => 'Search';

  @override
  String get totalBalance => 'Total Balance';

  @override
  String get requestPhysicalCard => 'Request Physical Card';

  @override
  String get cardInfo => 'Card info';

  @override
  String get addMoney => 'Add Money';

  @override
  String get promoTitle => 'Instant digital delivery';

  @override
  String get promoSubtitle =>
      'Gift cards & top-ups land in your wallet in seconds.';

  @override
  String get featured => 'Featured';

  @override
  String get navHome => 'Home';

  @override
  String get navCategories => 'Categories';

  @override
  String get navCart => 'Cart';

  @override
  String get navMenu => 'Menu';

  @override
  String get comingSoon => 'Coming soon.';

  @override
  String get loadFailed => 'Couldn\'t load products.';

  @override
  String get fromLabel => 'from';

  @override
  String get chooseAmount => 'Choose amount';

  @override
  String get quantityLabel => 'Quantity';

  @override
  String get requiredBadge => 'Required';

  @override
  String ratingsCount(int count) {
    return '($count ratings)';
  }

  @override
  String enterValue(String field) {
    return 'Enter $field';
  }

  @override
  String get deliveredInstantly =>
      'Credit is delivered to this account instantly.';

  @override
  String get addToCart => 'Add to cart';

  @override
  String get buyNow => 'Buy now';

  @override
  String get notifyMe => 'Notify me';

  @override
  String get totalLabel => 'Total';

  @override
  String get cartTitle => 'Your cart';

  @override
  String cartItemsCount(int count) {
    return '· $count items';
  }

  @override
  String get cartEmptyTitle => 'Your cart is empty';

  @override
  String get cartEmptySub => 'Browse the catalog and add gift cards & top-ups.';

  @override
  String get browseCatalog => 'Browse catalog';

  @override
  String get subtotalLabel => 'Subtotal';

  @override
  String get discountLabel => 'Discount';

  @override
  String get walletBalanceHint => 'Wallet balance:';

  @override
  String get checkoutCta => 'Checkout';

  @override
  String get checkoutTitle => 'Checkout';

  @override
  String get paymentFailed => 'Payment failed. Please try another method.';

  @override
  String get deliveryDetails => 'Delivery details';

  @override
  String get paymentMethodLabel => 'Payment method';

  @override
  String get payCardTitle => 'Card';

  @override
  String get payCardSub => 'Instant approval';

  @override
  String get payWalletTitle => 'Wallet';

  @override
  String get balanceLabel => 'Balance';

  @override
  String get insufficientBalance => 'Insufficient balance';

  @override
  String get payUsdtTitle => 'USDT';

  @override
  String get payUsdtSub => 'Auto-approve · Instant';

  @override
  String get promoCodePlaceholder => 'Promo code';

  @override
  String get applyLabel => 'Apply';

  @override
  String get placeOrderCta => 'Place order';

  @override
  String get fieldRequired => 'This field is required.';

  @override
  String get orderCompletedTitle => 'Order completed';

  @override
  String get orderProcessingTitle => 'We\'re completing your order';

  @override
  String get orderCompletedSub => 'Your codes have been delivered. Enjoy!';

  @override
  String get orderProcessingSub =>
      'Account-credit orders can take a few minutes.';

  @override
  String get orderLabel => 'Order';

  @override
  String get deliveredCodesLabel => 'Delivered codes';

  @override
  String get giftCardLabel => 'Gift card';

  @override
  String get copyLabel => 'Copy';

  @override
  String get copiedLabel => 'Copied';

  @override
  String get orderProcessingNote =>
      'We\'ll notify you the moment it\'s done. You can track its status from the order page.';

  @override
  String get viewOrderCta => 'View order';

  @override
  String get backToHomeCta => 'Back to home';

  @override
  String get orderItemsLabel => 'Items';

  @override
  String get statusPending => 'Pending';

  @override
  String get statusProcessing => 'Processing';

  @override
  String get statusCompleted => 'Completed';

  @override
  String get statusFailed => 'Failed';

  @override
  String get statusRefunded => 'Refunded';

  @override
  String get ordersTitle => 'Orders';

  @override
  String get filterAll => 'All';

  @override
  String get ordersEmptyTitle => 'No orders yet';

  @override
  String get ordersEmptySub =>
      'Your purchases will show up here once you place an order.';

  @override
  String get orderRefundedNote =>
      'This order was refunded. The amount was returned to your wallet.';

  @override
  String get orderTimelineLabel => 'Timeline';

  @override
  String get walletTitle => 'Wallet';

  @override
  String get currentBalance => 'Current balance';

  @override
  String get topUpCta => 'Top up';

  @override
  String get sendMoney => 'Send money';

  @override
  String get txHistory => 'Transaction history';

  @override
  String get walletEmptyTitle => 'No transactions yet';

  @override
  String get walletEmptySub =>
      'Top up your wallet to get started — your activity will show up here.';

  @override
  String get txTopUp => 'Top up';

  @override
  String get txPurchase => 'Purchase';

  @override
  String get txRefund => 'Refund';

  @override
  String get txAdjustment => 'Adjustment';

  @override
  String get topUpAmount => 'Top-up amount';

  @override
  String get topUpVia => 'Top up via';

  @override
  String topUpSuccess(String amount) {
    return '+$amount added to your wallet';
  }

  @override
  String get sendMoneyComingSoon => 'Send money is coming soon.';

  @override
  String get recipientLabel => 'Recipient (phone or email)';

  @override
  String get amountLabel => 'Amount';

  @override
  String get noteLabel => 'Note (optional)';

  @override
  String get sendCta => 'Send';

  @override
  String get darkMode => 'Dark mode';

  @override
  String get profileTitle => 'Profile';

  @override
  String get accountInfo => 'Account info';

  @override
  String get roleLabel => 'Role';

  @override
  String get savedPlayerIds => 'Saved player IDs';

  @override
  String get savedPlayerIdsEmptyTitle => 'No saved player IDs';

  @override
  String get savedPlayerIdsEmptySub =>
      'Save your player or account IDs for faster checkout.';

  @override
  String get addPlayerId => 'Add';

  @override
  String get playerIdHint => 'Enter a player ID';

  @override
  String get editProfile => 'Edit';

  @override
  String get cancel => 'Cancel';

  @override
  String get saveChanges => 'Save changes';

  @override
  String get profileSaved => 'Profile updated.';
}
