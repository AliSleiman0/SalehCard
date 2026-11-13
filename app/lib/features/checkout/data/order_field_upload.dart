import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_envelope.dart';
import '../../../core/network/providers.dart';

/// Uploads one checkout photo-field image (QR code, barcode …) and returns its
/// stored URL — the value submitted for that field on the order. Same
/// multipart `image` contract as the top-up proof upload.
typedef OrderFieldUploader = Future<String> Function(String filePath);

final orderFieldUploaderProvider = Provider<OrderFieldUploader>((ref) {
  final dio = ref.watch(dioProvider);
  return (filePath) async {
    final form = FormData.fromMap({
      'image': await MultipartFile.fromFile(filePath, filename: 'photo.jpg'),
    });
    final response = await dio.post<dynamic>('/orders/field-files', data: form);
    return (unwrap(response) as Map<String, dynamic>)['url'] as String;
  };
});
