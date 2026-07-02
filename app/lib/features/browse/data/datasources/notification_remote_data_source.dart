import 'package:dio/dio.dart';

import '../../../../core/network/api_envelope.dart';
import '../dtos/notification_dto.dart';

class NotificationRemoteDataSource {
  const NotificationRemoteDataSource(this._dio);

  final Dio _dio;

  /// GET /notifications → the caller's inbox, newest first. A single generous
  /// page — the app has no pagination UI (app-wide convention).
  Future<List<NotificationDto>> listNotifications() async {
    final response = await _dio.get<dynamic>(
      '/notifications',
      queryParameters: {'limit': 50},
    );
    final list = unwrap(response) as List<dynamic>? ?? const [];
    return [
      for (final item in list)
        NotificationDto.fromJson(item as Map<String, dynamic>),
    ];
  }

  /// GET /notifications/unread-count → the bell-badge number.
  Future<int> unreadCount() async {
    final response = await _dio.get<dynamic>('/notifications/unread-count');
    final data = unwrap(response) as Map<String, dynamic>? ?? const {};
    return (data['count'] as num?)?.toInt() ?? 0;
  }

  /// POST /notifications/read-all — marks the whole inbox read.
  Future<void> markAllRead() async {
    await _dio.post<dynamic>('/notifications/read-all');
  }

  /// POST /notifications/devices — registers this device's push token.
  Future<void> registerDevice(String token) async {
    await _dio.post<dynamic>(
      '/notifications/devices',
      data: {'token': token, 'platform': 'android'},
    );
  }

  /// DELETE /notifications/devices — removes this device's push token.
  Future<void> unregisterDevice(String token) async {
    await _dio.delete<dynamic>(
      '/notifications/devices',
      data: {'token': token},
    );
  }
}
