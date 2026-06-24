import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import '../config/app_config.dart';
import '../storage/token_store.dart';
import '../../features/auth/presentation/controllers/auth_controller.dart';
import 'dio_client.dart';
import 'interceptors/auth_interceptor.dart';

final secureStorageProvider = Provider<FlutterSecureStorage>(
  (ref) => const FlutterSecureStorage(),
);

final tokenStoreProvider = Provider<TokenStore>(
  (ref) => TokenStore(ref.watch(secureStorageProvider)),
);

final dioProvider = Provider<Dio>((ref) {
  final tokenStore = ref.watch(tokenStoreProvider);
  final interceptor = AuthInterceptor(
    tokenStore: tokenStore,
    baseUrl: AppConfig.apiBaseUrl,
    onSessionExpired: () =>
        ref.read(authControllerProvider.notifier).onSessionExpired(),
  );
  return createDio(
    baseUrl: AppConfig.apiBaseUrl,
    authInterceptor: interceptor,
  );
});
