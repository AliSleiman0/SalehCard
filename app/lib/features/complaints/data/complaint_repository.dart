import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_envelope.dart';
import '../../../core/network/providers.dart';
import '../domain/complaint.dart';

/// `/complaints` API (AuthRequired). Errors surface as Dio/Api exceptions;
/// screens map them with `mapError`.
class ComplaintRepository {
  ComplaintRepository(this._dio);

  final Dio _dio;

  Future<List<Complaint>> list() async {
    final response = await _dio.get<dynamic>(
      '/complaints',
      queryParameters: {'limit': 50},
    );
    return [
      for (final c in (unwrap(response) as List<dynamic>? ?? const []))
        Complaint.fromJson(c as Map<String, dynamic>),
    ];
  }

  Future<Complaint> get(String id) async {
    final response = await _dio.get<dynamic>('/complaints/$id');
    return Complaint.fromJson(unwrap(response) as Map<String, dynamic>);
  }

  Future<Complaint> create({
    required String category,
    required String subject,
    required String message,
    String? orderId,
    String? photoUrl,
  }) async {
    final response = await _dio.post<dynamic>(
      '/complaints',
      data: {
        'category': category,
        'subject': subject,
        'message': message,
        if (orderId != null && orderId.isNotEmpty) 'orderId': orderId,
        if (photoUrl != null && photoUrl.isNotEmpty) 'photoUrl': photoUrl,
      },
    );
    return Complaint.fromJson(unwrap(response) as Map<String, dynamic>);
  }

  Future<Complaint> reply(String id, String body) async {
    final response = await _dio.post<dynamic>(
      '/complaints/$id/messages',
      data: {'body': body},
    );
    return Complaint.fromJson(unwrap(response) as Map<String, dynamic>);
  }

  /// Uploads one photo (multipart `image`) → its stored URL.
  Future<String> uploadPhoto(String filePath) async {
    final form = FormData.fromMap({
      'image': await MultipartFile.fromFile(filePath, filename: 'photo.jpg'),
    });
    final response = await _dio.post<dynamic>('/complaints/photos', data: form);
    return (unwrap(response) as Map<String, dynamic>)['url'] as String;
  }
}

final complaintRepositoryProvider = Provider<ComplaintRepository>(
  (ref) => ComplaintRepository(ref.watch(dioProvider)),
);

/// The signed-in customer's complaints, newest first.
final complaintsProvider = FutureProvider.autoDispose<List<Complaint>>(
  (ref) => ref.watch(complaintRepositoryProvider).list(),
);

/// One complaint with its thread. Invalidated on a foreground complaint push
/// so an open thread updates live.
final complaintDetailProvider = FutureProvider.autoDispose
    .family<Complaint, String>(
      (ref, id) => ref.watch(complaintRepositoryProvider).get(id),
    );
