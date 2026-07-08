import 'package:dio/dio.dart';

import '../error/exceptions.dart';

/// Unwraps the standard API response envelope
/// `{success, data?, error?, meta?}` from a successful (2xx) [Response],
/// returning the `data` payload. Throws [ApiException] if the body signals an
/// error or has an unexpected shape.
///
/// Non-2xx responses never reach here — Dio raises [DioException] for those,
/// which repositories map via `mapError`.
dynamic unwrap(Response<dynamic> response) {
  final body = response.data;
  if (body is Map) {
    if (body['success'] == true) {
      return body['data'];
    }
    if (body['error'] is Map) {
      final error = body['error'] as Map;
      throw ApiException(
        code: error['code']?.toString() ?? 'UNKNOWN',
        message: error['message']?.toString() ?? 'Request failed',
        statusCode: response.statusCode,
      );
    }
  }
  throw ApiException(
    code: 'UNEXPECTED_RESPONSE',
    message: 'Unexpected response from server',
    statusCode: response.statusCode,
  );
}

/// Like [unwrap], but also returns the pagination `meta` block so paginated
/// callers can tell whether more pages exist. `meta` is null when the endpoint
/// omits it. Throws [ApiException] on the same error/shape conditions as [unwrap].
({dynamic data, Map<String, dynamic>? meta}) unwrapPaged(
  Response<dynamic> response,
) {
  final data = unwrap(response);
  final body = response.data;
  final meta = (body is Map && body['meta'] is Map)
      ? (body['meta'] as Map).cast<String, dynamic>()
      : null;
  return (data: data, meta: meta);
}
