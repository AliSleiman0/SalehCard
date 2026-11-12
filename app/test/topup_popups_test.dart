import 'package:flutter_test/flutter_test.dart';
import 'package:salehcard_app/features/browse/domain/entities/app_notification.dart';
import 'package:salehcard_app/features/shell/presentation/widgets/topup_notification_listener.dart';

AppNotification _n(String id, AppNotificationType type, DateTime at,
        {bool read = false}) =>
    AppNotification(
        id: id, title: id, body: '', createdAt: at, type: type, isRead: read);

void main() {
  test('pops only unread top-up notes newer than the baseline, oldest first',
      () {
    final since = DateTime.utc(2026, 11, 12, 10);
    final items = [
      _n('b', AppNotificationType.walletTopUpRejected,
          DateTime.utc(2026, 11, 12, 10, 5)),
      _n('a', AppNotificationType.walletTopUp, DateTime.utc(2026, 11, 12, 10, 1)),
      _n('old', AppNotificationType.walletTopUp, DateTime.utc(2026, 11, 12, 9, 59)),
      _n('read', AppNotificationType.walletTopUp,
          DateTime.utc(2026, 11, 12, 10, 2),
          read: true),
      _n('order', AppNotificationType.orderCompleted,
          DateTime.utc(2026, 11, 12, 10, 4)),
    ];
    expect(newTopUpPopups(items, since).map((n) => n.id), ['a', 'b']);
  });
}
