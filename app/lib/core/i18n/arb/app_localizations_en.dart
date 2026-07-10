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
  String get sendCodeButton => 'Send code';

  @override
  String get useCodeInstead => 'Sign in with a code instead';

  @override
  String get usePasswordInstead => 'Sign in with a password instead';

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
  String get offersEmpty => 'No offers right now. Check back soon.';

  @override
  String get productDetailUnavailable =>
      'This product is currently unavailable.';

  @override
  String get verifyChecking => 'Checking ID…';

  @override
  String get verifyFound => 'Account';

  @override
  String get verifyNotFound =>
      'We couldn\'t find that ID. Please check and try again.';

  @override
  String get verifyUnavailable =>
      'Couldn\'t verify right now — you can still continue.';

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
  String get nameLabel => 'Full name';

  @override
  String get nameHint => 'Your name';

  @override
  String get nameRequired => 'Please enter your name.';

  @override
  String get mobileNumberLabel => 'Mobile Number';

  @override
  String get phoneHint => '70 123 456';

  @override
  String get passwordHint => '••••••••';

  @override
  String get newPasswordHint => 'At least 8 characters';

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
  String get passwordTooShort => 'Password must be at least 8 characters.';

  @override
  String get passwordMismatch => 'Passwords do not match.';

  @override
  String get otpIncomplete => 'Enter the 6-digit code.';

  @override
  String get searchHint => 'Search';

  @override
  String get totalBalance => 'Total Balance';

  @override
  String get addMoney => 'Add Money';

  @override
  String get promoTitle => 'Instant digital delivery';

  @override
  String get promoSubtitle =>
      'Gift card codes delivered instantly after checkout.';

  @override
  String get featured => 'Featured';

  @override
  String get navHome => 'Home';

  @override
  String get navCategories => 'Categories';

  @override
  String get navCart => 'Cart';

  @override
  String get navOffers => 'Offers';

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
  String qtyTotalLine(int count, String unitPrice, String total) {
    return '$count × $unitPrice = $total';
  }

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
  String get creditAfterProcessing =>
      'Credit is added to this account after we process your order.';

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
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count items',
      one: '1 item',
    );
    return '· $_temp0';
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
  String get payWalletTitle => 'Wallet';

  @override
  String get balanceLabel => 'Balance';

  @override
  String get insufficientBalance => 'Insufficient balance';

  @override
  String get topUpWalletCta => 'Top up wallet';

  @override
  String get kycRequiredTitle => 'Verification required';

  @override
  String get kycRequiredBody =>
      'Verify your identity to place orders. It only takes a minute and is reviewed by our team.';

  @override
  String get kycRequiredCta => 'Verify identity';

  @override
  String get kycBannerBody =>
      'Verify your account to make purchases — it only takes a minute. Tap to start.';

  @override
  String get kycBannerPending =>
      'Your verification is being reviewed. You can purchase once it\'s approved.';

  @override
  String get promoCodePlaceholder => 'Promo code';

  @override
  String get applyLabel => 'Apply';

  @override
  String get placeOrderCta => 'Place order';

  @override
  String get fieldRequired => 'This field is required.';

  @override
  String get invalidLebanesePhone =>
      'Enter a valid Lebanese mobile number (e.g. 71 123 456).';

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
  String get topUpVia => 'How will you pay?';

  @override
  String get topUpNoteHint => 'Payment reference / note (optional)';

  @override
  String get topUpRequested =>
      'Request submitted — your wallet is credited once an admin confirms your payment.';

  @override
  String get topUpRequestsTitle => 'My top-up requests';

  @override
  String get statusApproved => 'Approved';

  @override
  String get statusRejected => 'Rejected';

  @override
  String walletPendingTitle(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count top-ups pending approval',
      one: 'Top-up pending approval',
    );
    return '$_temp0';
  }

  @override
  String get walletPendingHint =>
      'Balance updates once an admin confirms your payment.';

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
  String get loyaltyPoints => 'Loyalty points';

  @override
  String get verificationTitle => 'Verification';

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
  String get playerIdLabelHint => 'Label (e.g. PUBG main)';

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

  @override
  String get browseTitle => 'Categories';

  @override
  String categoryItemCount(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count items',
      one: '1 item',
      zero: 'No items',
    );
    return '$_temp0';
  }

  @override
  String get searchPromptTitle => 'Search the catalog';

  @override
  String get searchPromptSub => 'Type to find gift cards, top-ups & more.';

  @override
  String get searchEmptyTitle => 'No results';

  @override
  String get searchEmptySub => 'Try a different keyword or browse categories.';

  @override
  String get notificationsTitle => 'Notifications';

  @override
  String get notificationsEmptyTitle => 'You\'re all caught up';

  @override
  String get notificationsEmptySub =>
      'Notifications about your orders and wallet will show up here.';

  @override
  String get notifOrderCompletedTitle => 'Order completed';

  @override
  String notifOrderCompletedBody(String amount) {
    return 'Your order was delivered — $amount.';
  }

  @override
  String get notifOrderRefundedTitle => 'Order refunded';

  @override
  String notifOrderRefundedBody(String amount) {
    return '$amount was returned to your wallet.';
  }

  @override
  String get notifWalletTopUpTitle => 'Wallet topped up';

  @override
  String notifWalletTopUpBody(String amount) {
    return 'Your top-up of $amount was approved and added to your wallet.';
  }

  @override
  String get notifTopUpRejectedTitle => 'Top-up rejected';

  @override
  String notifTopUpRejectedBody(String amount) {
    return 'Your $amount top-up request was rejected.';
  }

  @override
  String get notifKycApprovedTitle => 'Identity verified';

  @override
  String get notifKycApprovedBody =>
      'Your account is verified — you can now purchase.';

  @override
  String get notifKycRejectedTitle => 'Verification rejected';

  @override
  String get notifKycRejectedBody => 'Please resubmit your information.';

  @override
  String get notifPromoTitle => 'Limited-time offer';

  @override
  String get notifPromoBody => 'Discover the latest gift cards and offers.';

  @override
  String get kycTitle => 'Verification';

  @override
  String get kycMenuLabel => 'Verification';

  @override
  String get kycBadgeUnverified => 'Not verified';

  @override
  String get kycBadgePending => 'Under review';

  @override
  String get kycBadgeVerified => 'Verified';

  @override
  String get kycBadgeRejected => 'Rejected';

  @override
  String get kycUnverifiedTitle => 'Verify your identity';

  @override
  String get kycUnverifiedBody =>
      'Verify your identity to unlock higher limits and faster checkout. It only takes a minute.';

  @override
  String get kycPendingTitle => 'Under review';

  @override
  String get kycPendingBody =>
      'We\'ve received your documents and our team is reviewing them. This usually takes a few minutes.';

  @override
  String get kycVerifiedTitle => 'You\'re verified';

  @override
  String get kycVerifiedBody =>
      'Your identity has been confirmed. All features are unlocked — enjoy higher limits and faster checkout.';

  @override
  String get kycRejectedTitle => 'Verification failed';

  @override
  String get kycRejectedBody =>
      'We couldn\'t verify your identity from the documents provided. Please check the details and resubmit.';

  @override
  String get kycVerifyNowCta => 'Verify now';

  @override
  String get kycResubmitCta => 'Resubmit';

  @override
  String get kycFormTitle => 'Identity verification';

  @override
  String get kycFullNameLabel => 'Full name';

  @override
  String get kycDobLabel => 'Date of birth';

  @override
  String get kycDobHint => 'Select your date of birth';

  @override
  String get kycPlaceOfBirthLabel => 'Place of birth';

  @override
  String get kycPlaceOfResidenceLabel => 'Place of residence';

  @override
  String get kycPlaceHint => 'City, country';

  @override
  String get kycDocTypeLabel => 'Document type';

  @override
  String get kycDocPassport => 'Passport';

  @override
  String get kycDocIdCard => 'ID card';

  @override
  String get kycDocLicense => 'Driver\'s license';

  @override
  String get kycDocNumberLabel => 'Document number';

  @override
  String get kycSubmitCta => 'Submit';

  @override
  String get kycFieldRequired => 'This field is required.';

  @override
  String get kycSubmittedSnack =>
      'Verification submitted — we\'ll review it shortly.';

  @override
  String get kycDocPhotosLabel => 'Document photos';

  @override
  String get kycDocFrontLabel => 'Front of document';

  @override
  String get kycDocBackLabel => 'Back of document';

  @override
  String get kycDocBackOptionalTag => 'Optional for passports';

  @override
  String get kycDocAddPhoto => 'Add photo';

  @override
  String get kycDocReplacePhoto => 'Replace photo';

  @override
  String get kycDocRemovePhoto => 'Remove';

  @override
  String get kycDocUploading => 'Uploading…';

  @override
  String get kycDocUploadFailed => 'Upload failed — tap to retry.';

  @override
  String get kycDocPhotoRequired => 'This photo is required.';

  @override
  String get kycDocSourceCamera => 'Take a photo';

  @override
  String get kycDocSourceGallery => 'Choose from gallery';

  @override
  String get writeReview => 'Write a review';

  @override
  String get reviewSheetTitle => 'Rate this product';

  @override
  String get reviewTapToRate => 'Tap a star to rate';

  @override
  String get reviewNoteOptionalHint => 'Tell us what went wrong (optional)';

  @override
  String get reviewSubmit => 'Submit review';

  @override
  String get reviewSubmittedPending =>
      'Thanks! Your review is pending approval.';

  @override
  String get reviewAlreadyReviewed => 'You\'ve already reviewed this product.';

  @override
  String get reviewYouReviewed => 'You reviewed this product';

  @override
  String get commonDone => 'Done';

  @override
  String get usdtPayLabel => 'Pay with USDT (crypto)';

  @override
  String get usdtDepositTitle => 'Send USDT';

  @override
  String get usdtSendExactly =>
      'Send exactly this amount — every digit matters, or your payment can\'t be matched automatically:';

  @override
  String get usdtAmountLabel => 'Amount';

  @override
  String get usdtAmountCopied => 'Amount copied';

  @override
  String usdtAddressLabel(String network) {
    return '$network deposit address';
  }

  @override
  String get usdtAddressCopied => 'Address copied';

  @override
  String usdtNetworkWarning(String network) {
    return 'Send only USDT on the $network network. Sending any other coin or network will lose the funds.';
  }

  @override
  String usdtExpiresIn(String time) {
    return 'Expires in $time';
  }

  @override
  String get usdtWaiting => 'Waiting for your payment…';

  @override
  String get usdtConfirming => 'Payment detected — confirming on-chain…';

  @override
  String get usdtConfirmedTitle => 'Payment confirmed';

  @override
  String get usdtConfirmedBody => 'Your USDT payment was confirmed.';

  @override
  String get usdtExpiredTitle => 'Payment window expired';

  @override
  String get usdtExpiredBody =>
      'No payment was received in time. Start again to get a fresh address.';

  @override
  String get usdtNetworkLabel => 'Network';
}
