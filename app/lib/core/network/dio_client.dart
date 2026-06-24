import 'package:dio/dio.dart';

import '../config/app_config.dart';

/// Builds the app-wide [Dio] instance: base URL, JSON, the native client
/// header, sane timeouts, and the supplied auth interceptor.
Dio createDio({
  required String baseUrl,
  required Interceptor authInterceptor,
}) {
  final dio = Dio(
    BaseOptions(
      baseUrl: baseUrl,
      contentType: Headers.jsonContentType,
      headers: {
        AppConfig.clientHeaderName: AppConfig.clientHeaderValue,
      },
      connectTimeout: const Duration(seconds: 15),
      receiveTimeout: const Duration(seconds: 20),
    ),
  );
  dio.interceptors.add(authInterceptor);
  return dio;
}
