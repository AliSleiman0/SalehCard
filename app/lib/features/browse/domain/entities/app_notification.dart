/// What a notification is about — drives the leading icon and the localized
/// copy. Unknown future server kinds fall back to [AppNotificationType.general],
/// which renders the raw server title/body.
enum AppNotificationType {
  orderCompleted,
  orderRefunded,
  walletTopUp,
  walletTopUpRejected,
  kycApproved,
  kycRejected,
  promo,
  general,
}

/// A single in-app notification (domain entity), as returned by
/// `GET /notifications`. [title]/[body] are the server's English fallback copy;
/// [data] carries structured values (orderId, amount, reason, ...) the UI uses
/// to compose localized strings for known [type]s.
class AppNotification {
  const AppNotification({
    required this.id,
    required this.title,
    required this.body,
    required this.createdAt,
    this.type = AppNotificationType.general,
    this.data = const {},
    this.isRead = true,
  });

  final String id;
  final String title;
  final String body;
  final DateTime createdAt;
  final AppNotificationType type;
  final Map<String, String> data;
  final bool isRead;
}
