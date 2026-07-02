import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../../domain/entities/app_notification.dart';
import '../../domain/repositories/notification_repository.dart';
import '../datasources/notification_remote_data_source.dart';

class NotificationRepositoryImpl implements NotificationRepository {
  const NotificationRepositoryImpl(this._remote);

  final NotificationRemoteDataSource _remote;

  @override
  Future<Either<Failure, List<AppNotification>>> listNotifications() async {
    try {
      final dtos = await _remote.listNotifications();
      return Right([for (final dto in dtos) dto.toEntity()]);
    } catch (error) {
      return Left(mapError(error));
    }
  }

  @override
  Future<Either<Failure, int>> unreadCount() async {
    try {
      return Right(await _remote.unreadCount());
    } catch (error) {
      return Left(mapError(error));
    }
  }

  @override
  Future<Either<Failure, void>> markAllRead() async {
    try {
      await _remote.markAllRead();
      return const Right(null);
    } catch (error) {
      return Left(mapError(error));
    }
  }

  @override
  Future<Either<Failure, void>> registerDevice(String token) async {
    try {
      await _remote.registerDevice(token);
      return const Right(null);
    } catch (error) {
      return Left(mapError(error));
    }
  }

  @override
  Future<Either<Failure, void>> unregisterDevice(String token) async {
    try {
      await _remote.unregisterDevice(token);
      return const Right(null);
    } catch (error) {
      return Left(mapError(error));
    }
  }
}
