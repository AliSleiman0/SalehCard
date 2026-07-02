import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../../domain/entities/app_notification.dart';
import '../../domain/repositories/notification_repository.dart';

/// Display-only stub for in-app notifications.
///
/// TODO(backend): replace stub with GET /notifications when an endpoint exists.
/// Returns a small static sample so the populated (default) state renders; the
/// screen localizes each row's title/body by [AppNotification.type], so the
/// English strings here are only a faithful fallback. This is the single wiring
/// point — swap this class for an HTTP implementation once the API exists.
class NotificationRepositoryStub implements NotificationRepository {
  const NotificationRepositoryStub();

  @override
  Future<Either<Failure, List<AppNotification>>> listNotifications() async {
    final now = DateTime.now();
    return Right(<AppNotification>[
      AppNotification(
        id: 'n1',
        type: AppNotificationType.orderCompleted,
        title: 'Order completed',
        body: 'Your codes have been delivered. Enjoy!',
        createdAt: now.subtract(const Duration(hours: 2)),
      ),
      AppNotification(
        id: 'n2',
        type: AppNotificationType.walletTopUp,
        title: 'Wallet topped up',
        body: 'Your top-up was approved and added to your wallet.',
        createdAt: now.subtract(const Duration(days: 1)),
      ),
      AppNotification(
        id: 'n3',
        type: AppNotificationType.promo,
        title: 'Limited-time offer',
        body: 'Discover the latest gift cards and offers.',
        createdAt: now.subtract(const Duration(days: 3)),
      ),
    ]);
  }
}
