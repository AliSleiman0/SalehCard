/// What a notification is about — drives the leading icon. Unknown future types
/// fall back to [AppNotificationType.general].
enum AppNotificationType { orderCompleted, walletTopUp, promo, general }

/// A single in-app notification (domain entity). Today these are produced by a
/// stub; the shape mirrors what a future `GET /notifications` would return.
class AppNotification {
  const AppNotification({
    required this.id,
    required this.title,
    required this.body,
    required this.createdAt,
    this.type = AppNotificationType.general,
  });

  final String id;
  final String title;
  final String body;
  final DateTime createdAt;
  final AppNotificationType type;
}
