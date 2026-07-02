import '../../domain/entities/app_notification.dart';

/// Hand-written DTO (no build_runner needed) for `GET /notifications` rows.
class NotificationDto {
  const NotificationDto({
    required this.id,
    this.kind = '',
    this.title = '',
    this.body = '',
    this.data = const {},
    this.readAt,
    this.createdAt,
  });

  final String id;
  final String kind;
  final String title;
  final String body;
  final Map<String, String> data;
  final String? readAt;
  final String? createdAt;

  factory NotificationDto.fromJson(Map<String, dynamic> json) =>
      NotificationDto(
        id: json['id'] as String? ?? '',
        kind: json['kind'] as String? ?? '',
        title: json['title'] as String? ?? '',
        body: json['body'] as String? ?? '',
        data:
            (json['data'] as Map<String, dynamic>?)?.map(
              (k, v) => MapEntry(k, v.toString()),
            ) ??
            const {},
        readAt: json['readAt'] as String?,
        createdAt: json['createdAt'] as String?,
      );

  AppNotification toEntity() => AppNotification(
    id: id,
    title: title,
    body: body,
    type: notificationTypeFromKind(kind),
    data: data,
    isRead: readAt != null,
    createdAt:
        (createdAt == null ? null : DateTime.tryParse(createdAt!)) ??
        DateTime.fromMillisecondsSinceEpoch(0),
  );
}

/// Maps a server kind string onto the app's notification type; unknown kinds
/// fall back to [AppNotificationType.general] (renders raw title/body).
AppNotificationType notificationTypeFromKind(String kind) {
  switch (kind) {
    case 'order_completed':
      return AppNotificationType.orderCompleted;
    case 'order_refunded':
      return AppNotificationType.orderRefunded;
    case 'topup_approved':
      return AppNotificationType.walletTopUp;
    case 'topup_rejected':
      return AppNotificationType.walletTopUpRejected;
    case 'kyc_approved':
      return AppNotificationType.kycApproved;
    case 'kyc_rejected':
      return AppNotificationType.kycRejected;
    default:
      return AppNotificationType.general;
  }
}
