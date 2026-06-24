import 'package:dio/dio.dart';

import '../../config/app_config.dart';
import '../../storage/token_store.dart';

/// Injects the Bearer access token on protected requests and transparently
/// refreshes it once on a 401, then replays the original request. If refresh
/// fails, [onSessionExpired] is invoked so the app can route to login.
///
/// Auth endpoints (`/auth/...`) are skipped for both injection and refresh to
/// avoid recursion. A dedicated [_refreshDio] (no interceptors) performs the
/// refresh call and the retry.
class AuthInterceptor extends Interceptor {
  AuthInterceptor({
    required this.tokenStore,
    required String baseUrl,
    required this.onSessionExpired,
  }) : _refreshDio = Dio(
          BaseOptions(
            baseUrl: baseUrl,
            headers: {
              AppConfig.clientHeaderName: AppConfig.clientHeaderValue,
            },
            contentType: Headers.jsonContentType,
          ),
        );

  final TokenStore tokenStore;
  final void Function() onSessionExpired;
  final Dio _refreshDio;

  bool _isAuthPath(String path) => path.contains('/auth/');

  @override
  void onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) async {
    if (!_isAuthPath(options.path)) {
      final access = await tokenStore.readAccess();
      if (access != null && access.isNotEmpty) {
        options.headers['Authorization'] = 'Bearer $access';
      }
    }
    handler.next(options);
  }

  @override
  void onError(
    DioException err,
    ErrorInterceptorHandler handler,
  ) async {
    final request = err.requestOptions;
    final alreadyRetried = request.extra['__retried'] == true;
    final shouldRefresh = err.response?.statusCode == 401 &&
        !_isAuthPath(request.path) &&
        !alreadyRetried;

    if (!shouldRefresh) {
      handler.next(err);
      return;
    }

    final refreshed = await _tryRefresh();
    if (!refreshed) {
      onSessionExpired();
      handler.next(err);
      return;
    }

    final access = await tokenStore.readAccess();
    request.headers['Authorization'] = 'Bearer $access';
    request.extra['__retried'] = true;
    try {
      final response = await _refreshDio.fetch<dynamic>(request);
      handler.resolve(response);
    } on DioException catch (retryError) {
      handler.next(retryError);
    }
  }

  Future<bool> _tryRefresh() async {
    final refresh = await tokenStore.readRefresh();
    if (refresh == null || refresh.isEmpty) return false;
    try {
      final response = await _refreshDio.post<dynamic>(
        '/auth/refresh',
        data: {'refreshToken': refresh},
      );
      final body = response.data;
      if (body is Map && body['success'] == true && body['data'] is Map) {
        final data = body['data'] as Map;
        await tokenStore.save(
          access: data['accessToken'] as String,
          refresh: data['refreshToken'] as String?,
        );
        return true;
      }
      return false;
    } catch (_) {
      return false;
    }
  }
}
