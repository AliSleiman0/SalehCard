import 'package:flutter_test/flutter_test.dart';
import 'package:salehcard_app/features/browse/data/dtos/notification_dto.dart';
import 'package:salehcard_app/features/browse/domain/entities/app_notification.dart';

void main() {
  group('notificationTypeFromKind', () {
    test('maps every server kind onto its type', () {
      expect(
        notificationTypeFromKind('order_completed'),
        AppNotificationType.orderCompleted,
      );
      expect(
        notificationTypeFromKind('order_refunded'),
        AppNotificationType.orderRefunded,
      );
      expect(
        notificationTypeFromKind('topup_approved'),
        AppNotificationType.walletTopUp,
      );
      expect(
        notificationTypeFromKind('topup_rejected'),
        AppNotificationType.walletTopUpRejected,
      );
      expect(
        notificationTypeFromKind('kyc_approved'),
        AppNotificationType.kycApproved,
      );
      expect(
        notificationTypeFromKind('kyc_rejected'),
        AppNotificationType.kycRejected,
      );
    });

    test('unknown kinds fall back to general', () {
      expect(
        notificationTypeFromKind('shiny_new_thing'),
        AppNotificationType.general,
      );
      expect(notificationTypeFromKind(''), AppNotificationType.general);
    });
  });

  group('NotificationDto.toEntity', () {
    test('parses a full row', () {
      final dto = NotificationDto.fromJson({
        'id': 'n1',
        'kind': 'topup_approved',
        'title': 'Top-up approved',
        'body': r'$25.00 was added to your wallet.',
        'data': {'topupId': 't1', 'amount': '25.00', 'channel': 'whish'},
        'createdAt': '2026-07-02T10:00:00Z',
      });
      final entity = dto.toEntity();
      expect(entity.type, AppNotificationType.walletTopUp);
      expect(entity.data['amount'], '25.00');
      expect(entity.isRead, isFalse); // no readAt → unread
      expect(entity.createdAt, DateTime.utc(2026, 7, 2, 10));
    });

    test('readAt marks the row read', () {
      final dto = NotificationDto.fromJson({
        'id': 'n2',
        'kind': 'kyc_approved',
        'readAt': '2026-07-02T11:00:00Z',
        'createdAt': '2026-07-02T10:00:00Z',
      });
      expect(dto.toEntity().isRead, isTrue);
    });

    test('tolerates missing optional fields', () {
      final entity = NotificationDto.fromJson({'id': 'n3'}).toEntity();
      expect(entity.type, AppNotificationType.general);
      expect(entity.data, isEmpty);
      expect(entity.isRead, isFalse);
    });
  });
}
