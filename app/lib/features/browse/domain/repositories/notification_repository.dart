import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../entities/app_notification.dart';

abstract interface class NotificationRepository {
  Future<Either<Failure, List<AppNotification>>> listNotifications();

  /// Number of unread notifications — feeds the bell badge.
  Future<Either<Failure, int>> unreadCount();

  /// Marks every notification read (called when the inbox screen opens).
  Future<Either<Failure, void>> markAllRead();

  /// Registers this device's FCM token for push delivery.
  Future<Either<Failure, void>> registerDevice(String token);

  /// Removes this device's FCM token (called on logout).
  Future<Either<Failure, void>> unregisterDevice(String token);
}
