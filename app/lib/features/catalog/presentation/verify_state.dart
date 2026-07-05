/// Lifecycle of a game-ID → nickname lookup on the product page. The lookup is
/// driven locally by the product screen (debounced); this file holds only the
/// state it renders.
enum VerifyStatus { idle, checking, found, notFound, unavailable }

class VerifyState {
  const VerifyState({
    this.status = VerifyStatus.idle,
    this.username = '',
    this.banned = false,
  });

  final VerifyStatus status;
  final String username;
  final bool banned;

  /// Whether the purchase may proceed: a resolved account, or a fail-open when
  /// the upstream check is unavailable. A not-found id (or a not-yet-checked
  /// field) blocks the Buy button.
  bool get allowsPurchase =>
      status == VerifyStatus.found || status == VerifyStatus.unavailable;
}
