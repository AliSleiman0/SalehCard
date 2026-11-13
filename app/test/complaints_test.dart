import 'package:flutter_test/flutter_test.dart';
import 'package:salehcard_app/core/push/push_service.dart';
import 'package:salehcard_app/features/browse/data/dtos/notification_dto.dart';
import 'package:salehcard_app/features/browse/domain/entities/app_notification.dart';
import 'package:salehcard_app/features/browse/presentation/screens/notifications_screen.dart';
import 'package:salehcard_app/features/complaints/domain/complaint.dart';

const _id = '6a1b2c3d4e5f60718293a4b5';

void main() {
  group('complaint push routing', () {
    test('a complaint reply / status push opens its thread', () {
      expect(
        pushRouteFor({
          'kind': 'complaint_reply',
          'complaintId': _id,
          'number': '1042',
        }),
        '/complaints/$_id',
      );
      expect(
        pushRouteFor({
          'kind': 'complaint_status',
          'complaintId': _id,
          'status': 'resolved',
        }),
        '/complaints/$_id',
      );
    });

    test('other kinds and malformed payloads do not deep-link', () {
      expect(pushRouteFor({'kind': 'order_completed', 'orderId': _id}), isNull);
      expect(pushRouteFor({'complaintId': _id}), isNull); // old server: no kind
      expect(pushRouteFor({'kind': 'complaint_reply'}), isNull);
      expect(
        pushRouteFor({'kind': 'complaint_reply', 'complaintId': '../evil'}),
        isNull,
      );
    });
  });

  group('complaint notifications in the inbox', () {
    test('kinds map onto the complaint types', () {
      expect(
        notificationTypeFromKind('complaint_reply'),
        AppNotificationType.complaintReply,
      );
      expect(
        notificationTypeFromKind('complaint_status'),
        AppNotificationType.complaintStatus,
      );
    });

    test('complaint rows are tappable, existing kinds stay informational', () {
      AppNotification row(String kind, Map<String, String> data) =>
          NotificationDto(id: 'n1', kind: kind, data: data).toEntity();
      expect(
        notificationRoute(row('complaint_reply', {'complaintId': _id})),
        '/complaints/$_id',
      );
      expect(
        notificationRoute(row('complaint_status', {'complaintId': _id})),
        '/complaints/$_id',
      );
      expect(
        notificationRoute(row('order_completed', {'orderId': _id})),
        isNull,
      );
      expect(notificationRoute(row('something_new', const {})), isNull);
    });
  });

  group('Complaint.fromJson', () {
    test('parses a thread, never exposing staff fields', () {
      final c = Complaint.fromJson({
        'id': _id,
        'ref': 'C-1042',
        'category': 'order',
        'orderId': 'abcdef0123456789abcdef01',
        'subject': 'Code not received',
        'status': 'in_progress',
        'updatedAt': '2026-11-13T10:00:00Z',
        'messages': [
          {
            'id': 'm1',
            'author': 'customer',
            'body': 'Hi',
            'photoUrl': 'https://x/uploads/complaints/a.jpg',
            'createdAt': '2026-11-13T09:00:00Z',
          },
          {
            'id': 'm2',
            'author': 'admin',
            'body': 'Looking',
            'createdAt': '2026-11-13T09:30:00Z',
          },
        ],
      });
      expect(c.ref, 'C-1042');
      expect(c.status, ComplaintStatus.inProgress);
      expect(c.isClosed, isFalse);
      expect(c.messages, hasLength(2));
      expect(c.messages.first.fromCustomer, isTrue);
      expect(c.messages.first.photoUrl, isNotNull);
      expect(c.messages.last.fromCustomer, isFalse);
      expect(c.messages.last.photoUrl, isNull);
    });

    test('list rows have no messages and unknown statuses degrade', () {
      final c = Complaint.fromJson({'id': _id, 'status': 'weird'});
      expect(c.messages, isEmpty);
      expect(c.status, ComplaintStatus.unknown);
      expect(complaintStatusFromString('closed'), ComplaintStatus.closed);
    });
  });
}
